package websocket

import (
	"log"

	"github.com/moura1001/websocket-colors-domination/server/model"

	"github.com/google/uuid"
)

type Pool struct {
	Register   chan *Client
	Unregister chan *Client
	clients    map[string]*Client
	Broadcast  chan Message
	games      map[string]*model.Game
}

func NewPool() *Pool {
	return &Pool{
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		clients:    map[string]*Client{},
		Broadcast:  make(chan Message),
		games:      map[string]*model.Game{},
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

			_, exist := message["method"]

			if exist {

				method, ok := message["method"].(string)
				if ok {
					pool.HandleClientMessage(method, message)
				}
			}
		}
	}
}

func (pool *Pool) HandleClientMessage(method string, message Message) {
	switch {
	case method == "create":
		clientId, ok := message["clientId"].(string)
		if ok {
			clientConn := pool.clients[clientId]
			if clientConn != nil {
				gameId := uuid.NewString()
				game := &model.Game{
					Id:      gameId,
					Cells:   uint8(16),
					Players: []model.Player{},
				}

				pool.games[gameId] = game

				content := BuildCreateMessage(game)
				clientConn.Conn.WriteJSON(content)
			}

		}
	}
}
