package websocket

import (
	"log"
)

type Pool struct {
	Register   chan *Client
	Unregister chan *Client
	clients    map[string]*Client
	Broadcast  chan Message
}

func NewPool() *Pool {
	return &Pool{
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		clients:    map[string]*Client{},
		Broadcast:  make(chan Message),
	}
}

func (pool *Pool) Start() {
	for {
		select {
		case client := <-pool.Register:
			if client.ID != "" {
				pool.clients[client.ID] = client
				log.Printf("client '%s' connected", client.ID)
				log.Println("Size of Connection Pool has been increased to: ", len(pool.clients))

				content := BuildConnectMessage(client.ID)
				client.Conn.WriteJSON(content)
			}
		case client := <-pool.Unregister:
			_, exist := pool.clients[client.ID]
			if exist {
				delete(pool.clients, client.ID)
				log.Printf("client '%s' disconnected", client.ID)
				log.Println("Size of Connection Pool has been decreased to: ", len(pool.clients))
			}
		case message := <-pool.Broadcast:
			log.Println("Received message: ", message)
			for clientId, client := range pool.clients {
				err := client.Conn.WriteMessage(1, []byte("Hi Client!"))
				if err != nil {
					log.Printf("problem send message to client '%s': '%v'\n", clientId, err)
				}
			}
		}
	}
}
