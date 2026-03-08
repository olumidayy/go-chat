package websocket

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	poolChannelBuffer = 256
	clientSendBuffer  = 32
	MaxClientsPerRoom = 20
)

type Pool struct {
	ID             string
	Register       chan *Client
	Unregister     chan *Client
	Clients        map[*Client]bool
	Broadcast      chan Message
	AddData        chan MessageData
	done           chan struct{}
	dictionary     map[string]struct{}
	letterSources  []string
	mu             sync.RWMutex
	GameInSession  bool
	currentLetters string
	foundWords     map[string]string
	playerScores   map[string]int
	playerWords    map[string]int
	gameTimer      *time.Timer
	activeNames    map[string]bool
	onEmpty        func() // called when last client leaves
}

func NewPool() *Pool {
	words := loadWords()

	return &Pool{
		Register:      make(chan *Client, poolChannelBuffer),
		Unregister:    make(chan *Client, poolChannelBuffer),
		Clients:       make(map[*Client]bool),
		Broadcast:     make(chan Message, poolChannelBuffer),
		AddData:       make(chan MessageData, poolChannelBuffer),
		done:          make(chan struct{}),
		dictionary:    buildWordSet(words),
		letterSources: buildLetterSources(words),
		GameInSession: false,
		activeNames:   make(map[string]bool),
	}
}

// Stop signals the Start goroutine to exit.
func (pool *Pool) Stop() {
	close(pool.done)
}

func (pool *Pool) Start() {
	for {
		select {
		case <-pool.done:
			log.Printf("pool %s: shutdown", pool.ID)
			return
		case client := <-pool.Register:
			// Enforce max clients per room
			if len(pool.Clients) >= MaxClientsPerRoom {
				client.Conn.Close()
				break
			}
			// Atomically check and reserve the name, then add to pool.Clients
			pool.mu.Lock()
			if client.Name != "" {
				if pool.activeNames[strings.ToLower(client.Name)] {
					pool.mu.Unlock()
					_ = client.Conn.WriteJSON(map[string]string{"error": "name_taken"})
					client.Conn.Close()
					break
				}
				pool.activeNames[strings.ToLower(client.Name)] = true
			}
			pool.Clients[client] = true
			pool.mu.Unlock()
			log.Printf("pool %s: %d clients", pool.ID, len(pool.Clients))

			// Announce join to everyone
			if client.Name != "" {
				members := pool.ActiveMembers()
				joinMsg := MessageData{
					Name:    "Jumble",
					Text:    fmt.Sprintf("%s joined the room.", client.Name),
					Members: members,
				}
				// If a game is in session, attach current state to the announcement
				pool.mu.RLock()
				if pool.GameInSession {
					joinMsg.Letters = pool.roundLettersLocked()
					joinMsg.Scores = pool.currentLeaderboardLocked()
					joinMsg.RoundState = roundStateActive
				}
				pool.mu.RUnlock()

				for c := range pool.Clients {
					select {
					case c.Send <- joinMsg:
					default:
					}
				}

				// If a game is in session, send a private catch-up to the new client
				pool.mu.RLock()
				if pool.GameInSession {
					catchUp := MessageData{
						Name:       "Jumble",
						Text:       fmt.Sprintf("A round is in progress! Form words from the letters above."),
						Scores:     pool.currentLeaderboardLocked(),
						Letters:    pool.roundLettersLocked(),
						RoundState: roundStateActive,
					}
					select {
					case client.Send <- catchUp:
					default:
					}
				}
				pool.mu.RUnlock()
			}
		case client := <-pool.Unregister:
			if _, ok := pool.Clients[client]; ok {
				leftName := client.Name
				delete(pool.Clients, client)
				close(client.Send)
				if client.Name != "" {
					pool.mu.Lock()
					delete(pool.activeNames, strings.ToLower(client.Name))
					pool.mu.Unlock()
				}
				// Announce departure
				if leftName != "" && len(pool.Clients) > 0 {
					members := pool.ActiveMembers()
					leaveMsg := MessageData{
						Name:    "Jumble",
						Text:    fmt.Sprintf("%s left the room.", leftName),
						Members: members,
					}
					pool.mu.RLock()
					if pool.GameInSession {
						leaveMsg.Letters = pool.roundLettersLocked()
						leaveMsg.Scores = pool.currentLeaderboardLocked()
						leaveMsg.RoundState = roundStateActive
					}
					pool.mu.RUnlock()
					for c := range pool.Clients {
						select {
						case c.Send <- leaveMsg:
						default:
						}
					}
				}
			}
			if len(pool.Clients) == 0 {
				pool.resetGame()
				if pool.onEmpty != nil {
					go pool.onEmpty()
				}
			}

		case message := <-pool.Broadcast:
			for client := range pool.Clients {
				select {
				case client.Send <- message.Body:
				default:
					log.Printf("pool %s: dropping slow client", pool.ID)
					delete(pool.Clients, client)
					close(client.Send)
					client.Conn.Close()
				}
			}
		case data := <-pool.AddData:
			if result, ok := pool.HandleGuess(data); ok {
				pool.Broadcast <- Message{Type: websocket.TextMessage, Body: result}
			}
		}
	}
}

func (pool *Pool) IsGameInSession() bool {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.GameInSession
}

// CurrentRoundState returns a catch-up message for the active round, or nil.
func (pool *Pool) CurrentRoundState() *MessageData {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	if !pool.GameInSession {
		return nil
	}
	msg := MessageData{
		Name:       "Jumble",
		Text:       "A round is already in progress! Keep guessing.",
		Scores:     pool.currentLeaderboardLocked(),
		Letters:    pool.roundLettersLocked(),
		RoundState: roundStateActive,
	}
	return &msg
}

func (pool *Pool) IsNameTaken(name string) bool {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return pool.activeNames[strings.ToLower(name)]
}

func (pool *Pool) ClientCount() int {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	return len(pool.activeNames)
}

// ActiveMembers returns the display names of all connected clients.
func (pool *Pool) ActiveMembers() []string {
	names := make([]string, 0, len(pool.Clients))
	for c := range pool.Clients {
		if c.Name != "" {
			names = append(names, c.Name)
		}
	}
	sort.Strings(names)
	return names
}

func (pool *Pool) StartGame(duration time.Duration, playerName string) (GameRound, bool) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	if pool.GameInSession {
		return GameRound{}, false
	}

	letters := shuffleLetters(pickLetterSource(pool.letterSources))

	pool.GameInSession = true
	pool.currentLetters = letters
	pool.foundWords = make(map[string]string)
	pool.playerScores = make(map[string]int)
	pool.playerWords = make(map[string]int)
	pool.registerPlayerLocked(playerName)
	if pool.gameTimer != nil {
		pool.gameTimer.Stop()
	}

	log.Printf("pool %s: game started", pool.ID)
	pool.gameTimer = time.AfterFunc(duration, func() {
		pool.endGame()
	})

	return GameRound{Letters: letters, Scores: pool.currentLeaderboardLocked()}, true
}

func (pool *Pool) HandleGuess(data MessageData) (MessageData, bool) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	if !pool.GameInSession || pool.currentLetters == "" {
		return MessageData{}, false
	}

	playerName := strings.TrimSpace(data.Name)
	if playerName == "" {
		playerName = "Someone"
	}
	pool.registerPlayerLocked(playerName)

	guess := normalizeWord(data.Text)
	if guess == "" || guess == "jumble" {
		return MessageData{}, false
	}

	if _, ok := pool.dictionary[guess]; !ok {
		return pool.roundMessageLocked(fmt.Sprintf("%s tried %s, but it is not in the dictionary.", playerName, strings.ToUpper(guess))), true
	}

	if !canBuildWord(guess, pool.currentLetters) {
		return pool.roundMessageLocked(fmt.Sprintf("%s tried %s, but it cannot be made from the current letters.", playerName, strings.ToUpper(guess))), true
	}

	if foundBy, found := pool.foundWords[guess]; found {
		return pool.roundMessageLocked(fmt.Sprintf("%s tried %s, but it was already found by %s.", playerName, strings.ToUpper(guess), foundBy)), true
	}

	winner := playerName

	pool.foundWords[guess] = winner
	earnedPoints := scoreWord(guess)
	pool.playerScores[winner] += earnedPoints
	pool.playerWords[winner]++
	totalScore := pool.playerScores[winner]
	wordsFound := pool.playerWords[winner]
	leaderboard := pool.currentLeaderboardLocked()

	return MessageData{
		Name:       "Jumble",
		Text:       fmt.Sprintf("%s found %s for %d Scrabble point%s.\nTotal: %d point%s across %d word%s.", winner, strings.ToUpper(guess), earnedPoints, pluralSuffix(earnedPoints), totalScore, pluralSuffix(totalScore), wordsFound, pluralSuffix(wordsFound)),
		Scores:     leaderboard,
		Letters:    pool.roundLettersLocked(),
		RoundState: roundStateActive,
	}, true
}

func (pool *Pool) endGame() {
	pool.mu.Lock()
	if !pool.GameInSession {
		pool.mu.Unlock()
		return
	}

	letters := spacedLetters(pool.currentLetters)
	wordsFound := len(pool.foundWords)
	winners, bestScore := determineWinners(pool.playerScores)
	leaderboardEntries := pool.currentLeaderboardLocked()
	leaderboard := formatLeaderboard(leaderboardEntries, 0)
	pool.resetGameLocked()
	pool.mu.Unlock()

	log.Printf("pool %s: game ended", pool.ID)
	lines := []string{
		"Time's up!",
	}
	if wordsFound > 0 {
		lines = append(lines,
			fmt.Sprintf("%d valid word%s found.", wordsFound, pluralSuffix(wordsFound)),
			formatWinnerLines(winners, bestScore),
		)
	} else {
		lines = append(lines, "No valid words were found.")
	}

	if leaderboard != "" {
		lines = append(lines, "Scores:", leaderboard)
	}

	pool.Broadcast <- Message{
		Type: websocket.TextMessage,
		Body: MessageData{
			Name:       "Jumble",
			Text:       strings.Join(lines, "\n"),
			Scores:     leaderboardEntries,
			Letters:    letters,
			RoundState: roundStateEnded,
		},
	}
}

func (pool *Pool) roundLettersLocked() string {
	if pool.currentLetters == "" {
		return ""
	}

	return spacedLetters(pool.currentLetters)
}

func (pool *Pool) roundMessageLocked(text string) MessageData {
	return MessageData{
		Name:       "Jumble",
		Text:       text,
		Scores:     pool.currentLeaderboardLocked(),
		Letters:    pool.roundLettersLocked(),
		RoundState: roundStateActive,
	}
}

func (pool *Pool) registerPlayerLocked(playerName string) {
	playerName = strings.TrimSpace(playerName)
	if playerName == "" {
		return
	}

	if pool.playerScores == nil {
		pool.playerScores = make(map[string]int)
	}

	if pool.playerWords == nil {
		pool.playerWords = make(map[string]int)
	}

	if _, ok := pool.playerScores[playerName]; !ok {
		pool.playerScores[playerName] = 0
	}

	if _, ok := pool.playerWords[playerName]; !ok {
		pool.playerWords[playerName] = 0
	}
}

func (pool *Pool) currentLeaderboardLocked() []LeaderboardEntry {
	return buildLeaderboard(pool.playerScores, pool.playerWords)
}

func (pool *Pool) resetGame() {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	pool.resetGameLocked()
}

func (pool *Pool) resetGameLocked() {
	pool.GameInSession = false
	pool.currentLetters = ""
	pool.foundWords = nil
	pool.playerScores = nil
	pool.playerWords = nil
	if pool.gameTimer != nil {
		pool.gameTimer.Stop()
		pool.gameTimer = nil
	}
}

func pluralSuffix(value int) string {
	if value == 1 {
		return ""
	}

	return "s"
}
