package websocket

import (
	"log"
	"time"

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
					pool.handleClientMessage(method, message)
				}
			}
		}
	}
}

func (pool *Pool) handleClientMessage(method string, message Message) {
	switch {
	case method == "create":
		clientId, ok := message["clientId"].(string)
		if ok {
			clientConn := pool.clients[clientId]
			if clientConn != nil {
				gameId := uuid.NewString()
				game := &model.Game{
					Id:         gameId,
					Cells:      uint8(16),
					Players:    []model.Player{},
					BoardState: map[uint8]string{},
				}

				pool.games[gameId] = game

				content := BuildCreateMessage(game)
				clientConn.Conn.WriteJSON(content)
			}

		}
	case method == "join":
		clientId, okClientId := message["clientId"].(string)
		gameId, okGameId := message["gameId"].(string)
		if okClientId && okGameId {
			game := pool.games[gameId]
			if game != nil && len(game.Players) < 3 {
				color := map[uint8]string{0: "red", 1: "green", 2: "blue"}[uint8(len(game.Players))]
				game.Players = append(game.Players, model.Player{
					ClientId: clientId,
					Color:    color,
				})

				content := BuildJoinMessage(game)
				// loop through all players and tell them that people has joined
				for _, player := range game.Players {
					pool.clients[player.ClientId].Conn.WriteJSON(content)
				}

				// start game
				if len(game.Players) == 3 {
					go pool.updateGameStateForPlayers(game)
				}
			}

		}
	case method == "play":
		clientId, okClientId := message["clientId"].(string)
		gameId, okGameId := message["gameId"].(string)
		cellId, okCellId := message["cellId"].(float64)
		if okClientId && okGameId && okCellId {
			var player *model.Player = nil
			game := pool.games[gameId]
			if game != nil {
				for _, p := range game.Players {
					if p.ClientId == clientId {
						player = &p
						break
					}
				}

				if player != nil {
					game.BoardState[uint8(cellId)] = player.Color
				}
			}
		}
	}
}

func (pool *Pool) updateGameStateForPlayers(game *model.Game) {
	for !game.IsFinished {

		time.Sleep(500 * time.Millisecond)

		content := BuildUpdateMessage(game)
		// loop through all players and send updated state of the game
		for _, player := range game.Players {
			pool.clients[player.ClientId].Conn.WriteJSON(content)
		}
	}
}
