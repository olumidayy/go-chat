package websocket

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	poolChannelBuffer = 256
	clientSendBuffer  = 32
)

type Pool struct {
	Register       chan *Client
	Unregister     chan *Client
	Clients        map[*Client]bool
	Broadcast      chan Message
	AddData        chan MessageData
	dictionary     map[string]struct{}
	letterSources  []string
	mu             sync.RWMutex
	GameInSession  bool
	currentLetters string
	foundWords     map[string]string
	playerScores   map[string]int
	playerWords    map[string]int
	gameTimer      *time.Timer
}

func NewPool() *Pool {
	words := loadWords()

	return &Pool{
		Register:      make(chan *Client, poolChannelBuffer),
		Unregister:    make(chan *Client, poolChannelBuffer),
		Clients:       make(map[*Client]bool),
		Broadcast:     make(chan Message, poolChannelBuffer),
		AddData:       make(chan MessageData, poolChannelBuffer),
		dictionary:    buildWordSet(words),
		letterSources: buildLetterSources(words),
		GameInSession: false,
	}
}

func (pool *Pool) Start() {
	for {
		select {
		case client := <-pool.Register:
			pool.Clients[client] = true
			fmt.Println("Size of Connection Pool: ", len(pool.Clients))
			for client := range pool.Clients {
				fmt.Println(client)
				// client.Conn.WriteJSON(Message{Type: 1, Body: "New User Joined..."})
			}
		case client := <-pool.Unregister:
			if _, ok := pool.Clients[client]; ok {
				delete(pool.Clients, client)
				close(client.Send)
			}
			if len(pool.Clients) == 0 {
				pool.resetGame()
			}
			fmt.Println("Size of Connection Pool: ", len(pool.Clients))
			for client := range pool.Clients {
				fmt.Println(client)
				// client.Conn.WriteJSON(Message{Type: 1, Body: "User Disconnected..."})
			}
		case message := <-pool.Broadcast:
			fmt.Println("Sending message to all clients in Pool")
			for client := range pool.Clients {
				select {
				case client.Send <- message.Body:
				default:
					fmt.Println("Client send buffer full. Disconnecting slow client.")
					delete(pool.Clients, client)
					close(client.Send)
					client.Conn.Close()
				}
			}
		case data := <-pool.AddData:
			fmt.Println("Data received in pool:", data)
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

	fmt.Println("TIMER STARTED!", time.Now())
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
	if guess == "" || guess == "anagrams" {
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
		Name:       "Anagrams",
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

	fmt.Println("TIMER EXPIRED!", time.Now())
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
			Name:       "Anagrams",
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
		Name:       "Anagrams",
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
