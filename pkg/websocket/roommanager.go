package websocket

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"sync"
)

const roomCodeLen = 6

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

// CreateRoom generates a unique room code, spins up a new Pool and returns the code.
func (rm *RoomManager) CreateRoom() (string, *Pool) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	code := rm.generateCodeLocked()
	pool := NewPool()
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

	return rm.rooms[strings.ToUpper(code)]
}

// RemoveRoom deletes a room from the manager and stops its pool goroutine.
func (rm *RoomManager) RemoveRoom(code string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	code = strings.ToUpper(code)
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
	for {
		code := make([]byte, roomCodeLen)
		for i := range code {
			n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
			code[i] = alphabet[n.Int64()]
		}
		codeStr := string(code)
		if _, exists := rm.rooms[codeStr]; !exists {
			return codeStr
		}
	}
}
