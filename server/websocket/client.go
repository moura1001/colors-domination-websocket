package websocket

import (
	"log"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID   string
	Conn *websocket.Conn
	Pool *Pool
}

func (c *Client) Read() {
	defer func() {
		c.Pool.Unregister <- c
		c.Conn.Close()
	}()

	for {
		msgType, msg, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("error reading from WebSocket client '%s'. Details: '%v'\n", c.ID, err)
			return
		}

		message := Message{Type: msgType, Body: string(msg)}
		c.Pool.Broadcast <- message
	}
}
