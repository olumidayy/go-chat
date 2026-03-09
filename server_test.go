package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/olumidayy/go-websockets/pkg/websocket"
)

func setupTestRoutes() {
	rm = websocket.NewRoomManager()
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Alice", "Alice"},
		{"  Bob  ", "Bob"},
		{"a\x00b", "ab"},
		{"a\x1Fb", "ab"},
		{"normal name", "normal name"},
	}
	for _, tt := range tests {
		got := sanitizeName(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSanitizeName_TruncatesLong(t *testing.T) {
	long := strings.Repeat("a", 100)
	got := sanitizeName(long)
	if len([]rune(got)) > maxUsernameLen {
		t.Errorf("sanitizeName should truncate to %d chars, got %d", maxUsernameLen, len([]rune(got)))
	}
}

func TestIsValidName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"Al", true},
		{"Alice", true},
		{"Bob_123", true},
		{"name-with-dash", true},
		{"name with space", true},
		{"Go 😀", true},
		{"😀😀", true},
		{"👨‍👩‍👧‍👦", true},
		{"", false},
		{"A", false},
		{"😀", false},
		{strings.Repeat("x", maxUsernameLen+1), false},
		{"<script>", false},
		{"name@home", false},
		{"hello!", false},
	}
	for _, tt := range tests {
		got := isValidName(tt.name)
		if got != tt.want {
			t.Errorf("isValidName(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestClientIP(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	ip := clientIP(req)
	if ip != "192.168.1.1" {
		t.Errorf("clientIP = %q, want 192.168.1.1", ip)
	}
}

func TestClientIP_NoPort(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.168.1.1"
	ip := clientIP(req)
	if ip != "192.168.1.1" {
		t.Errorf("clientIP = %q, want 192.168.1.1", ip)
	}
}

func TestRateLimiter_AllowsBurst(t *testing.T) {
	rl := newRateLimiter(1, 5)
	for i := 0; i < 5; i++ {
		if !rl.allow("10.0.0.1") {
			t.Errorf("request %d should be allowed within burst", i+1)
		}
	}
}

func TestRateLimiter_DeniesOverBurst(t *testing.T) {
	rl := newRateLimiter(1, 3)
	for i := 0; i < 3; i++ {
		rl.allow("10.0.0.1")
	}
	if rl.allow("10.0.0.1") {
		t.Error("request beyond burst should be denied")
	}
}

func TestRateLimiter_SeparateIPs(t *testing.T) {
	rl := newRateLimiter(1, 2)
	rl.allow("10.0.0.1")
	rl.allow("10.0.0.1")
	if rl.allow("10.0.0.1") {
		t.Error("IP1 should be rate-limited")
	}
	if !rl.allow("10.0.0.2") {
		t.Error("IP2 should not be rate-limited")
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := securityHeaders(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	handler(rr, req)

	expectedHeaders := []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
		"X-XSS-Protection",
		"Referrer-Policy",
		"Permissions-Policy",
		"Content-Security-Policy",
	}
	for _, h := range expectedHeaders {
		if rr.Header().Get(h) == "" {
			t.Errorf("missing security header: %s", h)
		}
	}
}

// Integration tests

func TestHandleCreateRoom_Success(t *testing.T) {
	setupTestRoutes()
	req := httptest.NewRequest(http.MethodPost, "/api/rooms/create", nil)
	req.RemoteAddr = "127.0.0.1:9999"
	rr := httptest.NewRecorder()

	handleCreateRoom(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	code, ok := resp["code"]
	if !ok || code == "" {
		t.Error("response should contain a non-empty 'code'")
	}
	if len(code) < websocket.MinRoomCodeLen || len(code) > websocket.MaxRoomCodeLen {
		t.Errorf("room code length = %d, want between %d and %d", len(code), websocket.MinRoomCodeLen, websocket.MaxRoomCodeLen)
	}
}

func TestHandleCreateRoom_WithDuration(t *testing.T) {
	setupTestRoutes()
	req := httptest.NewRequest(http.MethodPost, "/api/rooms/create", bytes.NewBufferString(`{"durationSeconds":90}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:9999"
	rr := httptest.NewRecorder()

	handleCreateRoom(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	code := resp["code"]
	if code == "" {
		t.Fatal("response should contain a room code")
	}

	pool := rm.GetRoom(code)
	if pool == nil {
		t.Fatal("created room was not found")
	}

	if got := pool.RoundDuration(); got != 90*time.Second {
		t.Fatalf("pool.RoundDuration() = %v, want %v", got, 90*time.Second)
	}
}

func TestHandleCreateRoom_InvalidDuration(t *testing.T) {
	setupTestRoutes()
	req := httptest.NewRequest(http.MethodPost, "/api/rooms/create", bytes.NewBufferString(`{"durationSeconds":20}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:9999"
	rr := httptest.NewRecorder()

	handleCreateRoom(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleCreateRoom_WithCustomCode(t *testing.T) {
	setupTestRoutes()
	req := httptest.NewRequest(http.MethodPost, "/api/rooms/create", bytes.NewBufferString(`{"durationSeconds":90,"roomCode":"myroom9"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:9999"
	rr := httptest.NewRecorder()

	handleCreateRoom(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if got := resp["code"]; got != "MYROOM9" {
		t.Fatalf("expected code MYROOM9, got %q", got)
	}
}

func TestHandleCreateRoom_InvalidCustomCode(t *testing.T) {
	setupTestRoutes()
	req := httptest.NewRequest(http.MethodPost, "/api/rooms/create", bytes.NewBufferString(`{"roomCode":"AB"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:9999"
	rr := httptest.NewRecorder()

	handleCreateRoom(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestHandleCreateRoom_CustomCodeTaken(t *testing.T) {
	setupTestRoutes()

	firstReq := httptest.NewRequest(http.MethodPost, "/api/rooms/create", bytes.NewBufferString(`{"roomCode":"MYROOM9"}`))
	firstReq.Header.Set("Content-Type", "application/json")
	firstReq.RemoteAddr = "127.0.0.1:9999"
	firstRR := httptest.NewRecorder()
	handleCreateRoom(firstRR, firstReq)
	if firstRR.Code != http.StatusOK {
		t.Fatalf("expected first request 200, got %d", firstRR.Code)
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/api/rooms/create", bytes.NewBufferString(`{"roomCode":"myroom9"}`))
	secondReq.Header.Set("Content-Type", "application/json")
	secondReq.RemoteAddr = "127.0.0.1:9999"
	secondRR := httptest.NewRecorder()
	handleCreateRoom(secondRR, secondReq)

	if secondRR.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", secondRR.Code)
	}
}

func TestHandleCreateRoom_MethodNotAllowed(t *testing.T) {
	setupTestRoutes()
	req := httptest.NewRequest(http.MethodGet, "/api/rooms/create", nil)
	rr := httptest.NewRecorder()

	handleCreateRoom(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestHandleFunFact_Success(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/funfact", nil)
	rr := httptest.NewRecorder()

	handleFunFact(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if strings.TrimSpace(resp["fact"]) == "" {
		t.Fatal("expected non-empty fact")
	}
}

func TestHandleFunFact_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/funfact", nil)
	rr := httptest.NewRecorder()

	handleFunFact(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestHandleEmojiOptions_Success(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/emojis", nil)
	rr := httptest.NewRecorder()

	handleEmojiOptions(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp struct {
		Emojis []map[string]string `json:"emojis"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	if len(resp.Emojis) == 0 {
		t.Fatal("expected non-empty emoji list")
	}
}

func TestHandleEmojiOptions_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/emojis", nil)
	rr := httptest.NewRecorder()

	handleEmojiOptions(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
}

func TestHandleCreateRoom_AtCapacity(t *testing.T) {
	setupTestRoutes()
	codes := make([]string, 0)
	for i := 0; i < maxRoomsGlobal; i++ {
		code, _ := rm.CreateRoom()
		codes = append(codes, code)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/rooms/create", nil)
	req.RemoteAddr = "127.0.0.1:9999"
	rr := httptest.NewRecorder()
	handleCreateRoom(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 at capacity, got %d", rr.Code)
	}

	for _, c := range codes {
		rm.RemoveRoom(c)
	}
}

func TestHandleGetRoom_Success(t *testing.T) {
	setupTestRoutes()
	code, _ := rm.CreateRoom()
	defer rm.RemoveRoom(code)

	req := httptest.NewRequest(http.MethodGet, "/api/rooms/"+code, nil)
	req.RemoteAddr = "127.0.0.1:9999"
	rr := httptest.NewRecorder()

	handleGetRoom(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var info websocket.RoomInfo
	if err := json.NewDecoder(rr.Body).Decode(&info); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if info.Code != code {
		t.Errorf("expected code %s, got %s", code, info.Code)
	}
}

func TestHandleGetRoom_NotFound(t *testing.T) {
	setupTestRoutes()
	req := httptest.NewRequest(http.MethodGet, "/api/rooms/ZZZZZZ", nil)
	req.RemoteAddr = "127.0.0.1:9999"
	rr := httptest.NewRecorder()

	handleGetRoom(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestHandleGetRoom_InvalidCode(t *testing.T) {
	setupTestRoutes()
	req := httptest.NewRequest(http.MethodGet, "/api/rooms/AB", nil)
	req.RemoteAddr = "127.0.0.1:9999"
	rr := httptest.NewRecorder()

	handleGetRoom(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for short code, got %d", rr.Code)
	}
}

func TestHandleGetRoom_MethodNotAllowed(t *testing.T) {
	setupTestRoutes()
	req := httptest.NewRequest(http.MethodPost, "/api/rooms/ABCDEF", nil)
	rr := httptest.NewRecorder()

	handleGetRoom(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rr.Code)
	}
}

func TestServeWs_MissingParams(t *testing.T) {
	setupTestRoutes()
	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	req.RemoteAddr = "127.0.0.1:9999"
	rr := httptest.NewRecorder()

	serveWs(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing params, got %d", rr.Code)
	}
}

func TestServeWs_InvalidName(t *testing.T) {
	setupTestRoutes()
	code, _ := rm.CreateRoom()
	defer rm.RemoveRoom(code)

	req := httptest.NewRequest(http.MethodGet, "/ws?room="+code+"&name=<script>", nil)
	req.RemoteAddr = "127.0.0.1:9999"
	rr := httptest.NewRecorder()

	serveWs(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid name, got %d", rr.Code)
	}
}

func TestServeWs_RoomNotFound(t *testing.T) {
	setupTestRoutes()
	req := httptest.NewRequest(http.MethodGet, "/ws?room=ZZZZZZ&name=Alice", nil)
	req.RemoteAddr = "127.0.0.1:9999"
	rr := httptest.NewRecorder()

	serveWs(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestCreateThenGetRoom_Integration(t *testing.T) {
	setupTestRoutes()

	createReq := httptest.NewRequest(http.MethodPost, "/api/rooms/create", nil)
	createReq.RemoteAddr = "127.0.0.1:9999"
	createRR := httptest.NewRecorder()
	handleCreateRoom(createRR, createReq)

	if createRR.Code != http.StatusOK {
		t.Fatalf("create: expected 200, got %d", createRR.Code)
	}

	var createResp map[string]string
	if err := json.NewDecoder(createRR.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	code := createResp["code"]

	getReq := httptest.NewRequest(http.MethodGet, "/api/rooms/"+code, nil)
	getReq.RemoteAddr = "127.0.0.1:9999"
	getRR := httptest.NewRecorder()
	handleGetRoom(getRR, getReq)

	if getRR.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d: %s", getRR.Code, getRR.Body.String())
	}

	var info websocket.RoomInfo
	if err := json.NewDecoder(getRR.Body).Decode(&info); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if info.Code != code {
		t.Errorf("get: expected code %s, got %s", code, info.Code)
	}
	if info.Players != 0 {
		t.Errorf("get: expected 0 players, got %d", info.Players)
	}

	rm.RemoveRoom(code)
}
