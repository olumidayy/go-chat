package main

import (
	"encoding/json"
	"fmt"
	"html"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/olumidayy/go-websockets/pkg/websocket"
)

var rm *websocket.RoomManager

// ─── Security constants ─────────────────────────────────────
const (
	maxUsernameLen = 24
	minUsernameLen = 2
	maxMessageLen  = 300
	maxRoomsGlobal = 100
)

var validUsername = regexp.MustCompile(`^[a-zA-Z0-9 _\-]+$`)

// ─── Rate limiter (per-IP token bucket) ─────────────────────
type rateLimiter struct {
	mu      sync.Mutex
	clients map[string]*bucket
	rate    float64 // tokens per second
	burst   int
}

type bucket struct {
	tokens    float64
	lastCheck time.Time
}

func newRateLimiter(rate float64, burst int) *rateLimiter {
	rl := &rateLimiter{
		clients: make(map[string]*bucket),
		rate:    rate,
		burst:   burst,
	}
	// Periodic cleanup of stale entries
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			rl.mu.Lock()
			stale := time.Now().Add(-10 * time.Minute)
			for ip, b := range rl.clients {
				if b.lastCheck.Before(stale) {
					delete(rl.clients, ip)
				}
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.clients[ip]
	now := time.Now()
	if !ok {
		rl.clients[ip] = &bucket{tokens: float64(rl.burst) - 1, lastCheck: now}
		return true
	}
	elapsed := now.Sub(b.lastCheck).Seconds()
	b.lastCheck = now
	b.tokens += elapsed * rl.rate
	if b.tokens > float64(rl.burst) {
		b.tokens = float64(rl.burst)
	}
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// Separate limiters for different endpoints
var (
	apiLimiter = newRateLimiter(2, 10) // 2 req/s, burst 10 — API calls
	wsLimiter  = newRateLimiter(1, 5)  // 1 req/s, burst 5  — WS upgrades
)

// ─── Security headers middleware ────────────────────────────
func securityHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline'; "+
				"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; "+
				"font-src 'self' https://fonts.gstatic.com; "+
				"img-src 'self' data:; "+
				"connect-src 'self' ws: wss:; "+
				"frame-ancestors 'none'")
		next(w, r)
	}
}

// ─── IP extraction ──────────────────────────────────────────
func clientIP(r *http.Request) string {
	// Trust X-Forwarded-For only behind a known proxy; for now use RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// ─── Input validation ───────────────────────────────────────
func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	// Strip any HTML
	name = html.UnescapeString(name)
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 { // strip control characters
			return -1
		}
		return r
	}, name)
	// Truncate to max length
	if utf8.RuneCountInString(name) > maxUsernameLen {
		runes := []rune(name)
		name = string(runes[:maxUsernameLen])
	}
	return strings.TrimSpace(name)
}

func isValidName(name string) bool {
	l := utf8.RuneCountInString(name)
	return l >= minUsernameLen && l <= maxUsernameLen && validUsername.MatchString(name)
}

// ─── Handlers ───────────────────────────────────────────────
func serveWs(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)
	if !wsLimiter.allow(ip) {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	roomCode := strings.TrimSpace(r.URL.Query().Get("room"))
	name := sanitizeName(r.URL.Query().Get("name"))

	if roomCode == "" || name == "" {
		http.Error(w, "room and name are required", http.StatusBadRequest)
		return
	}

	if !isValidName(name) {
		http.Error(w, "invalid name", http.StatusBadRequest)
		return
	}

	pool := rm.GetRoom(roomCode)
	if pool == nil {
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	if pool.IsNameTaken(name) {
		conn, err := websocket.Upgrade(w, r)
		if err != nil {
			http.Error(w, "upgrade failed", http.StatusInternalServerError)
			return
		}
		_ = conn.WriteJSON(map[string]string{"error": "name_taken"})
		conn.Close()
		return
	}

	conn, err := websocket.Upgrade(w, r)
	if err != nil {
		http.Error(w, "upgrade failed", http.StatusInternalServerError)
		return
	}

	client := websocket.NewClient(conn, pool, name)
	pool.Register <- client
	go client.Write()
	client.Read()
}

func handleCreateRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ip := clientIP(r)
	if !apiLimiter.allow(ip) {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	if rm.RoomCount() >= maxRoomsGlobal {
		http.Error(w, "server at capacity", http.StatusServiceUnavailable)
		return
	}

	code, _ := rm.CreateRoom()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"code": code})
}

func handleGetRoom(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ip := clientIP(r)
	if !apiLimiter.allow(ip) {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	// Path: /api/rooms/XXXXXX
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/rooms/"), "/")
	code := strings.TrimSpace(parts[0])
	if code == "" || len(code) != 6 {
		http.Error(w, "invalid room code", http.StatusBadRequest)
		return
	}

	pool := rm.GetRoom(code)
	if pool == nil {
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(websocket.RoomInfo{
		Code:    strings.ToUpper(code),
		Players: pool.ClientCount(),
	})
}

func setupRoutes() {
	rm = websocket.NewRoomManager()

	http.HandleFunc("/ws", serveWs)
	http.HandleFunc("/api/rooms/create", securityHeaders(handleCreateRoom))
	http.HandleFunc("/api/rooms/", securityHeaders(handleGetRoom))

	http.HandleFunc("/favicon.svg", securityHeaders(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=86400")
		http.ServeFile(w, r, "favicon.svg")
	}))

	http.HandleFunc("/", securityHeaders(func(w http.ResponseWriter, r *http.Request) {
		// Only serve the root path to avoid path traversal
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "websockets.html")
	}))
}

func main() {
	fmt.Println("Distributed Chat App v0.01")
	setupRoutes()
	server := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		fmt.Println(err)
	}
}
