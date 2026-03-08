package websocket

import (
	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

const (
	maxMessageSize = 2048 // max bytes per incoming WS message (allows for UTF-8 and JSON overhead)
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return false // require an Origin header
		}
		parsed, err := url.Parse(origin)
		if err != nil {
			return false
		}
		// Allow same-origin only: compare parsed host to request host
		return parsed.Host == r.Host
	},
}

func Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return nil, err
	}
	// Set maximum message size the server will read
	conn.SetReadLimit(maxMessageSize)
	return conn, nil
}
