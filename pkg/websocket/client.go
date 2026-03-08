package websocket

import (
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID   string
	Name string
	Conn *websocket.Conn
	Pool *Pool
	Send chan MessageData
	mu   sync.Mutex
}

type Message struct {
	Type int `json:"type"`
	Body MessageData
}

type MessageData struct {
	Name       string             `json:"name"`
	Text       string             `json:"text"`
	Scores     []LeaderboardEntry `json:"scores,omitempty"`
	Letters    string             `json:"letters,omitempty"`
	RoundState string             `json:"roundState,omitempty"`
	Members    []string           `json:"members,omitempty"`
}

const (
	writeWait        = 10 * time.Second
	pongWait         = 60 * time.Second
	pingPeriod       = (pongWait * 9) / 10 // Must be < pongWait
	maxTextLen       = 200                 // max characters in a chat message
	roundStateActive = "active"
	roundStateEnded  = "ended"
)

func NewClient(conn *websocket.Conn, pool *Pool, name string) *Client {
	return &Client{
		Name: name,
		Conn: conn,
		Pool: pool,
		Send: make(chan MessageData, clientSendBuffer),
	}
}

// sanitizeText strips control chars, trims, and caps length.
// Does NOT HTML-escape — the frontend handles display escaping with escapeHTML().
func sanitizeText(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' {
			return -1
		}
		return r
	}, s)
	if utf8.RuneCountInString(s) > maxTextLen {
		runes := []rune(s)
		s = string(runes[:maxTextLen])
	}
	return s
}

func (c *Client) Read() {
	defer func() {
		c.Pool.Unregister <- c
		c.Conn.Close()
	}()

	// Set read deadline; extended by pong handler
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		messageType, p, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Println("Read error:", err)
			}
			return
		}

		var data MessageData
		err = json.Unmarshal(p, &data)
		if err != nil {
			log.Println("Unmarshal error:", err)
			return
		}

		// ── Server-side validation ──────────────────────────
		// Force the authenticated name (prevent spoofing)
		data.Name = c.Name
		// Sanitize the message text
		data.Text = sanitizeText(data.Text)
		// Drop empty messages
		if data.Text == "" {
			continue
		}

		if c.Pool.IsGameInSession() {
			c.Pool.AddData <- data
		}

		message := Message{Type: messageType, Body: data}
		c.Pool.Broadcast <- message

		if strings.EqualFold(strings.TrimSpace(data.Text), "ANAGRAMS") {
			if round, started := c.Pool.StartGame(defaultGameDuration, data.Name); started {
				letters := spacedLetters(round.Letters)
				c.Pool.Broadcast <- Message{
					Type: websocket.TextMessage,
					Body: MessageData{Name: "Anagrams", Text: "Game started.\nRound ends in 1 minute.", Scores: round.Scores, Letters: letters, RoundState: roundStateActive},
				}
				c.Pool.Broadcast <- Message{
					Type: websocket.TextMessage,
					Body: MessageData{Name: "Anagrams", Text: "Form words using some or all of these letters.", Scores: round.Scores, Letters: letters, RoundState: roundStateActive},
				}
			} else if c.Pool.IsGameInSession() {
				// Game already running — nudge the player
				if state := c.Pool.CurrentRoundState(); state != nil {
					select {
					case c.Send <- *state:
					default:
					}
				}
			}
		}
	}
}

func (c *Client) Write() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				// Channel closed — send close frame
				c.mu.Lock()
				c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				c.mu.Unlock()
				return
			}
			if err := c.writeJSON(message); err != nil {
				log.Println("Write error:", err)
				return
			}
		case <-ticker.C:
			// Send ping to keep connection alive and detect dead clients
			c.mu.Lock()
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			err := c.Conn.WriteMessage(websocket.PingMessage, nil)
			c.mu.Unlock()
			if err != nil {
				return
			}
		}
	}
}

func (c *Client) writeJSON(message MessageData) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.Conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return err
	}

	return c.Conn.WriteJSON(message)
}
