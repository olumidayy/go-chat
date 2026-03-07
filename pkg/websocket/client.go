package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID   string
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
}

const (
	writeWait        = 10 * time.Second
	roundStateActive = "active"
	roundStateEnded  = "ended"
)

func NewClient(conn *websocket.Conn, pool *Pool) *Client {
	return &Client{
		Conn: conn,
		Pool: pool,
		Send: make(chan MessageData, clientSendBuffer),
	}
}

func (c *Client) Read() {
	defer func() {
		c.Pool.Unregister <- c
		c.Conn.Close()
	}()

	for {
		messageType, p, err := c.Conn.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			return
		}

		var data MessageData
		err = json.Unmarshal([]byte(p), &data)
		if err != nil {
			log.Println("Unmarshal error:", err)
			return
		}

		if c.Pool.IsGameInSession() {
			c.Pool.AddData <- data
		}
		fmt.Println("Received:", data)

		message := Message{Type: messageType, Body: data}
		c.Pool.Broadcast <- message
		fmt.Printf("Message Received: %+v\n", message)

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
			}
		}
	}
}

func (c *Client) Write() {
	defer c.Conn.Close()

	for message := range c.Send {
		if err := c.writeJSON(message); err != nil {
			log.Println("Write error:", err)
			return
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
