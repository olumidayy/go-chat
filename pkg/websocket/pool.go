package websocket

import (
	"fmt"
	"time"
)

type Pool struct {
	Register   chan *Client
	Unregister chan *Client
	Clients    map[*Client]bool
	Broadcast  chan Message
	AddData  chan MessageData
	StartGame chan bool
	GameInSession  bool
}

func NewPool() *Pool {
	return &Pool{
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan Message),
		AddData:  make(chan MessageData),
		StartGame:  make(chan bool),
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
			break
		case client := <-pool.Unregister:
			delete(pool.Clients, client)
			fmt.Println("Size of Connection Pool: ", len(pool.Clients))
			for client := range pool.Clients {
				fmt.Println(client)
				// client.Conn.WriteJSON(Message{Type: 1, Body: "User Disconnected..."})
			}
			break
		case message := <-pool.Broadcast:
			fmt.Println("Sending message to all clients in Pool")
			for client := range pool.Clients {
				if err := client.Conn.WriteJSON(message.Body); err != nil {
					fmt.Println(err)
					return
				}
			}
		case val := <-pool.StartGame:
			fmt.Println("A game is starting")
			for client := range pool.Clients {
				fmt.Println(client)
				if !val { client.Pool.GameInSession = val; return }
				go time.AfterFunc(10 * time.Second, func() {
					client.Pool.GameInSession = val
				})
				continue
			}
		}
	}
}
