package websocket

import (
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

const (
	maxMessageSize = 512 // max bytes per incoming WS message
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return false // require an Origin header
		}
		host := r.Host
		// Allow same-origin only: origin must end with the request host
		// e.g. http://localhost:8080 matches Host: localhost:8080
		return strings.HasSuffix(origin, "://"+host)
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
