package websocket

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNewRoomManager(t *testing.T) {
	rm := NewRoomManager()
	if rm == nil {
		t.Fatal("NewRoomManager returned nil")
	}
	if rm.RoomCount() != 0 {
		t.Errorf("new RoomManager should have 0 rooms, got %d", rm.RoomCount())
	}
}

func TestCreateRoom(t *testing.T) {
	rm := NewRoomManager()
	code, pool := rm.CreateRoom()
	defer rm.RemoveRoom(code)

	if code == "" {
		t.Error("CreateRoom returned empty code")
	}
	if len(code) != roomCodeLen {
		t.Errorf("room code length = %d, want %d", len(code), roomCodeLen)
	}
	if pool == nil {
		t.Fatal("CreateRoom returned nil pool")
	}
	if rm.RoomCount() != 1 {
		t.Errorf("expected 1 room, got %d", rm.RoomCount())
	}
}

func TestCreateRoomWithDuration(t *testing.T) {
	rm := NewRoomManager()
	code, pool := rm.CreateRoomWithDuration(90 * time.Second)
	defer rm.RemoveRoom(code)

	if pool == nil {
		t.Fatal("CreateRoomWithDuration returned nil pool")
	}

	if got := pool.RoundDuration(); got != 90*time.Second {
		t.Fatalf("pool.RoundDuration() = %v, want %v", got, 90*time.Second)
	}
}

func TestIsValidRoomCode(t *testing.T) {
	valid := []string{"ABCDE", "ROOM123", "MYCODE10", "abc123"}
	for _, code := range valid {
		if !IsValidRoomCode(code) {
			t.Fatalf("expected %q to be valid", code)
		}
	}

	invalid := []string{"", "ABCD", "ABCDEFGHIJK", "AB-CD", "AB_CD", "AB CD", "🧩ROOM"}
	for _, code := range invalid {
		if IsValidRoomCode(code) {
			t.Fatalf("expected %q to be invalid", code)
		}
	}
}

func TestCreateRoomWithCode_Success(t *testing.T) {
	rm := NewRoomManager()
	code, pool, err := rm.CreateRoomWithCode("alpha9", 45*time.Second)
	if err != nil {
		t.Fatalf("CreateRoomWithCode returned error: %v", err)
	}
	defer rm.RemoveRoom(code)

	if code != "ALPHA9" {
		t.Fatalf("expected normalized code ALPHA9, got %s", code)
	}
	if pool == nil {
		t.Fatal("CreateRoomWithCode returned nil pool")
	}
	if got := pool.RoundDuration(); got != 45*time.Second {
		t.Fatalf("pool.RoundDuration() = %v, want %v", got, 45*time.Second)
	}
}

func TestCreateRoomWithCode_Invalid(t *testing.T) {
	rm := NewRoomManager()
	_, _, err := rm.CreateRoomWithCode("AB", time.Minute)
	if !errors.Is(err, ErrInvalidRoomCode) {
		t.Fatalf("expected ErrInvalidRoomCode, got %v", err)
	}
}

func TestCreateRoomWithCode_Taken(t *testing.T) {
	rm := NewRoomManager()
	code, _, err := rm.CreateRoomWithCode("ROOM77", time.Minute)
	if err != nil {
		t.Fatalf("first CreateRoomWithCode failed: %v", err)
	}
	defer rm.RemoveRoom(code)

	_, _, err = rm.CreateRoomWithCode("room77", time.Minute)
	if !errors.Is(err, ErrRoomCodeTaken) {
		t.Fatalf("expected ErrRoomCodeTaken, got %v", err)
	}
}

func TestCreateRoom_UniqueCodes(t *testing.T) {
	rm := NewRoomManager()
	codes := make(map[string]bool)
	for i := 0; i < 20; i++ {
		code, _ := rm.CreateRoom()
		defer rm.RemoveRoom(code)
		if codes[code] {
			t.Errorf("duplicate room code: %s", code)
		}
		codes[code] = true
	}
}

func TestGetRoom_Found(t *testing.T) {
	rm := NewRoomManager()
	code, pool := rm.CreateRoom()
	defer rm.RemoveRoom(code)

	got := rm.GetRoom(code)
	if got != pool {
		t.Error("GetRoom should return the same pool")
	}
}

func TestGetRoom_NotFound(t *testing.T) {
	rm := NewRoomManager()
	if rm.GetRoom("ZZZZZZ") != nil {
		t.Error("GetRoom should return nil for unknown codes")
	}
}

func TestRemoveRoom(t *testing.T) {
	rm := NewRoomManager()
	code, _ := rm.CreateRoom()
	rm.RemoveRoom(code)
	if rm.RoomCount() != 0 {
		t.Errorf("expected 0 rooms after RemoveRoom, got %d", rm.RoomCount())
	}
	if rm.GetRoom(code) != nil {
		t.Error("removed room should not be accessible")
	}
}

func TestRemoveRoom_NonExistent(t *testing.T) {
	rm := NewRoomManager()
	rm.RemoveRoom("DOESNT")
}

func TestRoomCount(t *testing.T) {
	rm := NewRoomManager()
	if rm.RoomCount() != 0 {
		t.Error("new manager should have 0 rooms")
	}
	c1, _ := rm.CreateRoom()
	c2, _ := rm.CreateRoom()
	if rm.RoomCount() != 2 {
		t.Errorf("expected 2 rooms, got %d", rm.RoomCount())
	}
	rm.RemoveRoom(c1)
	if rm.RoomCount() != 1 {
		t.Errorf("expected 1 room after removal, got %d", rm.RoomCount())
	}
	rm.RemoveRoom(c2)
}

func TestListRooms_Empty(t *testing.T) {
	rm := NewRoomManager()
	list := rm.ListRooms()
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d items", len(list))
	}
}

func TestListRooms(t *testing.T) {
	rm := NewRoomManager()
	c1, _ := rm.CreateRoom()
	c2, _ := rm.CreateRoom()
	defer rm.RemoveRoom(c1)
	defer rm.RemoveRoom(c2)

	list := rm.ListRooms()
	if len(list) != 2 {
		t.Fatalf("expected 2 rooms, got %d", len(list))
	}
	codes := map[string]bool{}
	for _, info := range list {
		codes[info.Code] = true
	}
	if !codes[c1] || !codes[c2] {
		t.Errorf("ListRooms missing expected codes: want %s and %s, got %v", c1, c2, list)
	}
}

func TestRoomManager_ConcurrentCreateRemove(t *testing.T) {
	rm := NewRoomManager()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			code, _ := rm.CreateRoom()
			rm.RemoveRoom(code)
		}()
	}
	wg.Wait()
	if rm.RoomCount() != 0 {
		t.Errorf("expected 0 rooms after concurrent create/remove, got %d", rm.RoomCount())
	}
}
