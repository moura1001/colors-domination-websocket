package websocket

import (
	"encoding/json"
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
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("error reading from client '%s'. Details: '%v'\n", c.ID, err)
			return
		}

		var message Message
		err = json.Unmarshal(msg, &message)

		if err != nil {
			log.Printf("error decoding message '%s' from client '%s'. Details: '%v'\n", string(msg), c.ID, err)
			return
		}

		c.Pool.Broadcast <- message
	}
}
