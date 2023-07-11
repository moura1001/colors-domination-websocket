package websocket

import (
	"log"
)

type Pool struct {
	Register       chan *Client
	Unregister     chan *Client
	Clients        map[string]*Client
	Broadcast      chan Message
	messageHandler OnReceivedMessageHandler
}

func NewPool(messageHandler OnReceivedMessageHandler) *Pool {
	return &Pool{
		Register:       make(chan *Client),
		Unregister:     make(chan *Client),
		Clients:        map[string]*Client{},
		Broadcast:      make(chan Message),
		messageHandler: messageHandler,
	}
}

func (pool *Pool) Start() {
	for {
		select {
		case client := <-pool.Register:
			if client.ID != "" {
				pool.Clients[client.ID] = client
				log.Printf("client '%s' connected", client.ID)
				log.Println("Size of Connection Pool has been increased to: ", len(pool.Clients))

				content := BuildConnectMessage(client.ID)
				client.Conn.WriteJSON(content)
			}
		case client := <-pool.Unregister:
			_, exist := pool.Clients[client.ID]
			if exist {
				delete(pool.Clients, client.ID)
				log.Printf("client '%s' disconnected", client.ID)
				log.Println("Size of Connection Pool has been decreased to: ", len(pool.Clients))
			}
		case message := <-pool.Broadcast:
			log.Println("Received message: ", message)

			_, exist := message["method"]

			if exist {

				method, ok := message["method"].(string)
				if ok {
					go pool.messageHandler(method, message)
				}
			}
		}
	}
}
