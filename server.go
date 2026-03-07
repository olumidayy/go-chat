package main

import (
	"fmt"
	"net/http"

	"github.com/olumidayy/go-websockets/pkg/websocket"
)

func serveWs(pool *websocket.Pool, w http.ResponseWriter, r *http.Request) {
	fmt.Println("WebSocket Endpoint Hit")
	conn, err := websocket.Upgrade(w, r)
	if err != nil {
		fmt.Fprintf(w, "%+v\n", err)
		return
	}

	client := websocket.NewClient(conn, pool)

	pool.Register <- client
	go client.Write()
	client.Read()
}

func setupRoutes() {
	pool := websocket.NewPool()
	go pool.Start()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(pool, w, r)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "websockets.html")
	})
}

func main() {
	fmt.Println("Distributed Chat App v0.01")
	setupRoutes()
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println(err)
	}
}
