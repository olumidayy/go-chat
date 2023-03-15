module github.com/olumidayy/go-websockets

go 1.20

require github.com/gorilla/websocket v1.5.0 // indirect

require (
	websocket v1.0.0
)

replace (
	websocket v1.0.0 => ./pkg/websocket
)