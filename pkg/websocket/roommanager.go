package websocket

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
)

const (
	roomCodeLen    = 6
	MinRoomCodeLen = 5
	MaxRoomCodeLen = 10
)

var (
	ErrInvalidRoomCode = errors.New("invalid room code")
	ErrRoomCodeTaken   = errors.New("room code already exists")
)

// RoomManager holds all active rooms keyed by their room code.
type RoomManager struct {
	mu    sync.RWMutex
	rooms map[string]*Pool
}

func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms: make(map[string]*Pool),
	}
}

func NormalizeRoomCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func IsValidRoomCode(code string) bool {
	code = NormalizeRoomCode(code)
	if len(code) < MinRoomCodeLen || len(code) > MaxRoomCodeLen {
		return false
	}

	for _, r := range code {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		return false
	}

	return true
}

// CreateRoom generates a unique room code, spins up a new Pool and returns the code.
func (rm *RoomManager) CreateRoom() (string, *Pool) {
	return rm.CreateRoomWithDuration(defaultGameDuration)
}

// CreateRoomWithDuration generates a unique room code with a configured round duration.
func (rm *RoomManager) CreateRoomWithDuration(duration time.Duration) (string, *Pool) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	code := rm.generateCodeLocked()
	return rm.createRoomLocked(code, duration)
}

// CreateRoomWithCode creates a room with a caller-provided code.
func (rm *RoomManager) CreateRoomWithCode(code string, duration time.Duration) (string, *Pool, error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	code = NormalizeRoomCode(code)
	if !IsValidRoomCode(code) {
		return "", nil, ErrInvalidRoomCode
	}

	if _, exists := rm.rooms[code]; exists {
		return "", nil, ErrRoomCodeTaken
	}

	createdCode, pool := rm.createRoomLocked(code, duration)
	return createdCode, pool, nil
}

func (rm *RoomManager) createRoomLocked(code string, duration time.Duration) (string, *Pool) {
	pool := NewPool()
	pool.SetRoundDuration(duration)
	pool.ID = code
	pool.onEmpty = func() {
		rm.RemoveRoom(code)
	}
	rm.rooms[code] = pool
	go pool.Start()

	fmt.Printf("Room created: %s (total rooms: %d)\n", code, len(rm.rooms))
	return code, pool
}

// GetRoom returns the pool for a room code, or nil if not found.
func (rm *RoomManager) GetRoom(code string) *Pool {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return rm.rooms[NormalizeRoomCode(code)]
}

// RemoveRoom deletes a room from the manager and stops its pool goroutine.
func (rm *RoomManager) RemoveRoom(code string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	code = NormalizeRoomCode(code)
	if pool, ok := rm.rooms[code]; ok {
		pool.Stop()
		delete(rm.rooms, code)
		fmt.Printf("Room removed: %s (total rooms: %d)\n", code, len(rm.rooms))
	}
}

// RoomCount returns the number of active rooms.
func (rm *RoomManager) RoomCount() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return len(rm.rooms)
}

// ListRooms returns a snapshot of all active room codes and their player counts.
func (rm *RoomManager) ListRooms() []RoomInfo {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	list := make([]RoomInfo, 0, len(rm.rooms))
	for code, pool := range rm.rooms {
		list = append(list, RoomInfo{
			Code:    code,
			Players: pool.ClientCount(),
		})
	}
	return list
}

type RoomInfo struct {
	Code    string `json:"code"`
	Players int    `json:"players"`
}

func (rm *RoomManager) generateCodeLocked() string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no I/O/0/1 to avoid confusion
generate:
	for {
		code := make([]byte, roomCodeLen)
		for i := range code {
			n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
			if err != nil {
				fmt.Printf("failed to generate random room code: %v\n", err)
				continue generate
			}
			code[i] = alphabet[n.Int64()]
		}
		codeStr := NormalizeRoomCode(string(code))
		if _, exists := rm.rooms[codeStr]; !exists {
			return codeStr
		}
	}
}
