package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("problem upgrading connection to WebSocket: '%v'\n", err)
	}

	log.Println("client connected")

	err = conn.WriteMessage(1, []byte("Hi Client!"))
	if err != nil {
		log.Printf("problem send message to client: '%v'\n", err)
	}

	reader(conn)
}

func reader(conn *websocket.Conn) {
	for {
		_, msg, err := conn.ReadMessage()

		if err != nil {
			log.Printf("error reading from WebSocket: '%v'\n", err)
			return
		}

		fmt.Printf("ws received message: '%s'\n", string(msg))
	}
}
