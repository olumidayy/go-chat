package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

func check(e error) {
	if e != nil {
		log.Fatal(e)
		return
	}
}

type Client struct {
	ID   string
	Conn *websocket.Conn
	Pool *Pool
	mu   sync.Mutex
}

type Message struct {
	Type int `json:"type"`
	Body MessageData
}

type MessageData struct {
	Name string `json:"name"`
	Text string `json:"text"`
}

func (c *Client) Read() {
	defer func() {
		c.Pool.Unregister <- c
		c.Conn.Close()
	}()
	mydir, err := os.Getwd()
	check(err)
	fmt.Println(mydir)

	path := filepath.Join(mydir, "pkg/websocket/words.txt")
	b, err := os.ReadFile(path)
	check(err)

	str := string(b)
	words := strings.Split(str, "   ")
	fmt.Println(words[0:20])

	for {
		messageType, p, err := c.Conn.ReadMessage()
		check(err)

		var data MessageData
		err = json.Unmarshal([]byte(p), &data)
		check(err)

		if c.Pool.GameInSession {
			c.Pool.AddData <- data
		}
		fmt.Println(data)

		message := Message{Type: messageType, Body: data}
		c.Pool.Broadcast <- message
		fmt.Printf("Message Received: %+v\n", message)

		if strings.ToUpper(data.Text) == "ANAGRAMS" && !c.Pool.GameInSession {
			data = MessageData{Name: "Anagrams", Text: "Game started. It'll end in 1 minute."}
			c.Pool.Broadcast <- Message{Type: messageType, Body: data}
			c.Pool.GameInSession = true
			go CountDown(c)
		}
	}
}

func CountDown(c *Client) {
	fmt.Println("TIMER STARTED!", time.Now())
	time.AfterFunc(1 * time.Minute, func() {
		fmt.Println("TIMER EXPIRED!", time.Now())
		for i := range c.Pool.AddData {
			fmt.Println(i)
		}
		c.Pool.GameInSession = false
	})
}
