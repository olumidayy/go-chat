package websocket

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func newTestPool() *Pool {
	dict := map[string]struct{}{
		"art":       {},
		"rat":       {},
		"tar":       {},
		"trade":     {},
		"read":      {},
		"dear":      {},
		"dare":      {},
		"tread":     {},
		"adventure": {},
		"note":      {},
		"tone":      {},
		"book":      {},
	}
	sources := []string{"adventure", "paintwork"}

	return &Pool{
		Register:      make(chan *Client, poolChannelBuffer),
		Unregister:    make(chan *Client, poolChannelBuffer),
		Clients:       make(map[*Client]bool),
		Broadcast:     make(chan Message, poolChannelBuffer),
		AddData:       make(chan MessageData, poolChannelBuffer),
		done:          make(chan struct{}),
		dictionary:    dict,
		letterSources: sources,
		GameInSession: false,
		activeNames:   make(map[string]bool),
	}
}

func TestIsGameInSession_InitFalse(t *testing.T) {
	pool := newTestPool()
	if pool.IsGameInSession() {
		t.Error("new pool should not have a game in session")
	}
}

func TestStartGame_Success(t *testing.T) {
	pool := newTestPool()
	round, started := pool.StartGame(50*time.Millisecond, "Alice")
	if !started {
		t.Fatal("StartGame should succeed on first call")
	}
	if round.Letters == "" {
		t.Error("StartGame should return non-empty letters")
	}
	if !pool.IsGameInSession() {
		t.Error("game should be in session after StartGame")
	}
	pool.resetGame()
}

func TestStartGame_RejectsDuplicate(t *testing.T) {
	pool := newTestPool()
	_, started := pool.StartGame(time.Second, "Alice")
	if !started {
		t.Fatal("first StartGame should succeed")
	}
	_, started2 := pool.StartGame(time.Second, "Bob")
	if started2 {
		t.Error("second StartGame should return false while game is active")
	}
	pool.resetGame()
}

func TestStartGame_RegistersPlayer(t *testing.T) {
	pool := newTestPool()
	pool.StartGame(time.Second, "Alice")
	pool.mu.RLock()
	_, ok := pool.playerScores["Alice"]
	pool.mu.RUnlock()
	if !ok {
		t.Error("StartGame should register the starting player's score")
	}
	pool.resetGame()
}

func TestHandleGuess_NoGameInSession(t *testing.T) {
	pool := newTestPool()
	_, ok := pool.HandleGuess(MessageData{Name: "Alice", Text: "art"})
	if ok {
		t.Error("HandleGuess should return false when no game is in session")
	}
}

func TestHandleGuess_ValidWord(t *testing.T) {
	pool := newTestPool()
	pool.StartGame(5*time.Second, "Alice")
	pool.mu.Lock()
	pool.currentLetters = "ADVENTURE"
	pool.mu.Unlock()

	result, ok := pool.HandleGuess(MessageData{Name: "Alice", Text: "art"})
	if !ok {
		t.Fatal("HandleGuess should accept a valid word")
	}
	if !strings.Contains(result.Text, "found") {
		t.Errorf("expected 'found' in result text, got: %s", result.Text)
	}
	pool.resetGame()
}

func TestHandleGuess_NotInDictionary(t *testing.T) {
	pool := newTestPool()
	pool.StartGame(5*time.Second, "Alice")
	pool.mu.Lock()
	pool.currentLetters = "ADVENTURE"
	pool.mu.Unlock()

	result, ok := pool.HandleGuess(MessageData{Name: "Alice", Text: "xyz"})
	if !ok {
		t.Fatal("HandleGuess should return a message for unknown words")
	}
	if !strings.Contains(result.Text, "not in the dictionary") {
		t.Errorf("expected 'not in the dictionary', got: %s", result.Text)
	}
	pool.resetGame()
}

func TestHandleGuess_CannotBuildFromLetters(t *testing.T) {
	pool := newTestPool()
	pool.StartGame(5*time.Second, "Alice")
	pool.mu.Lock()
	pool.currentLetters = "BOOK"
	pool.mu.Unlock()

	result, ok := pool.HandleGuess(MessageData{Name: "Alice", Text: "note"})
	if !ok {
		t.Fatal("HandleGuess should return a message")
	}
	if !strings.Contains(result.Text, "cannot be made") {
		t.Errorf("expected 'cannot be made', got: %s", result.Text)
	}
	pool.resetGame()
}

func TestHandleGuess_DuplicateWord(t *testing.T) {
	pool := newTestPool()
	pool.StartGame(5*time.Second, "Alice")
	pool.mu.Lock()
	pool.currentLetters = "ADVENTURE"
	pool.mu.Unlock()

	pool.HandleGuess(MessageData{Name: "Alice", Text: "art"})
	result, ok := pool.HandleGuess(MessageData{Name: "Bob", Text: "art"})
	if !ok {
		t.Fatal("HandleGuess should return a message for duplicate words")
	}
	if !strings.Contains(result.Text, "already found") {
		t.Errorf("expected 'already found', got: %s", result.Text)
	}
	pool.resetGame()
}

func TestHandleGuess_EmptyGuessIgnored(t *testing.T) {
	pool := newTestPool()
	pool.StartGame(5*time.Second, "Alice")
	_, ok := pool.HandleGuess(MessageData{Name: "Alice", Text: ""})
	if ok {
		t.Error("empty guess should be ignored")
	}
	pool.resetGame()
}

func TestHandleGuess_JumbleKeywordIgnored(t *testing.T) {
	pool := newTestPool()
	pool.StartGame(5*time.Second, "Alice")
	_, ok := pool.HandleGuess(MessageData{Name: "Alice", Text: "jumble"})
	if ok {
		t.Error("'jumble' keyword should be ignored as a guess")
	}
	pool.resetGame()
}

func TestHandleGuess_IncrementsScore(t *testing.T) {
	pool := newTestPool()
	pool.StartGame(5*time.Second, "Alice")
	pool.mu.Lock()
	pool.currentLetters = "ADVENTURE"
	pool.mu.Unlock()

	pool.HandleGuess(MessageData{Name: "Alice", Text: "art"})

	pool.mu.RLock()
	score := pool.playerScores["Alice"]
	words := pool.playerWords["Alice"]
	pool.mu.RUnlock()

	if score <= 0 {
		t.Errorf("expected positive score, got %d", score)
	}
	if words != 1 {
		t.Errorf("expected 1 word found, got %d", words)
	}
	pool.resetGame()
}

func TestCurrentRoundState_NilWhenNoGame(t *testing.T) {
	pool := newTestPool()
	if pool.CurrentRoundState() != nil {
		t.Error("CurrentRoundState should be nil when no game is in session")
	}
}

func TestCurrentRoundState_NonNilDuringGame(t *testing.T) {
	pool := newTestPool()
	pool.StartGame(time.Second, "Alice")
	state := pool.CurrentRoundState()
	if state == nil {
		t.Fatal("CurrentRoundState should not be nil during a game")
	}
	if state.RoundState != roundStateActive {
		t.Errorf("expected roundState=%q, got %q", roundStateActive, state.RoundState)
	}
	pool.resetGame()
}

func TestIsNameTaken(t *testing.T) {
	pool := newTestPool()
	pool.mu.Lock()
	pool.activeNames["alice"] = true
	pool.mu.Unlock()

	if !pool.IsNameTaken("Alice") {
		t.Error("IsNameTaken should be case-insensitive and find 'Alice'")
	}
	if pool.IsNameTaken("Bob") {
		t.Error("IsNameTaken should return false for unknown names")
	}
}

func TestClientCount_Empty(t *testing.T) {
	pool := newTestPool()
	if pool.ClientCount() != 0 {
		t.Error("new pool should have 0 clients")
	}
}

func TestActiveMembers_Empty(t *testing.T) {
	pool := newTestPool()
	members := pool.ActiveMembers()
	if len(members) != 0 {
		t.Error("new pool should have 0 active members")
	}
}

func TestResetGame(t *testing.T) {
	pool := newTestPool()
	pool.StartGame(time.Second, "Alice")
	pool.resetGame()
	if pool.IsGameInSession() {
		t.Error("game should not be in session after resetGame")
	}
}

func TestEndGame_TimerTriggersEnd(t *testing.T) {
	pool := newTestPool()
	pool.StartGame(50*time.Millisecond, "Alice")

	pool.mu.Lock()
	pool.currentLetters = "ADVENTURE"
	pool.mu.Unlock()

	time.Sleep(150 * time.Millisecond)

	if pool.IsGameInSession() {
		t.Error("game should have ended after timer elapsed")
	}

	select {
	case msg := <-pool.Broadcast:
		if msg.Body.RoundState != roundStateEnded {
			t.Errorf("expected roundState=%q in end message, got %q", roundStateEnded, msg.Body.RoundState)
		}
	case <-time.After(time.Second):
		t.Error("expected end-game broadcast message")
	}
}

func TestPool_ConcurrentGuesses(t *testing.T) {
	pool := newTestPool()
	pool.StartGame(5*time.Second, "Alice")
	pool.mu.Lock()
	pool.currentLetters = "ADVENTURE"
	pool.mu.Unlock()

	var wg sync.WaitGroup
	words := []string{"art", "rat", "tar", "trade", "read", "dear", "dare", "tread"}
	for i, w := range words {
		wg.Add(1)
		name := "Player" + itoa(i)
		go func(word, player string) {
			defer wg.Done()
			pool.HandleGuess(MessageData{Name: player, Text: word})
		}(w, name)
	}
	wg.Wait()
	pool.resetGame()
}
