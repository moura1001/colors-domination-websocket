package server

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/moura1001/websocket-colors-domination/server/model"
	ws "github.com/moura1001/websocket-colors-domination/server/websocket"
)

type Server struct {
	pool  *ws.Pool
	games map[string]*model.Game
}

func NewServer() *Server {
	server := new(Server)
	server.games = map[string]*model.Game{}
	server.pool = ws.NewPool(server.handleClientMessage)

	return server
}

func serveWs(pool *ws.Pool, w http.ResponseWriter, r *http.Request) {
	log.Println("WebSocket endpoint hit")
	conn, err := ws.Upgrade(w, r)
	if err != nil {
		fmt.Fprintf(w, "%+v\n", err)
		return
	}

	client := &ws.Client{
		ID:   uuid.NewString(),
		Conn: conn,
		Pool: pool,
	}

	pool.Register <- client
	client.Read()
}

func (server *Server) SetupRoutes() {
	fs := http.FileServer(http.Dir("./client/static"))
	http.Handle("/", fs)

	go server.pool.Start()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(server.pool, w, r)
	})
}

func (server *Server) handleClientMessage(method string, message ws.Message) {
	switch {
	case method == "create":
		clientId, ok := message["clientId"].(string)
		if ok {
			clientConn := server.pool.Clients[clientId]
			if clientConn != nil {
				gameId := uuid.NewString()
				game := &model.Game{
					Id:         gameId,
					Cells:      uint8(16),
					Players:    []model.Player{},
					BoardState: map[uint8]string{},
				}

				server.games[gameId] = game

				content := ws.BuildCreateMessage(game)
				clientConn.Conn.WriteJSON(content)
			}

		}
	case method == "join":
		clientId, okClientId := message["clientId"].(string)
		gameId, okGameId := message["gameId"].(string)
		if okClientId && okGameId {
			game := server.games[gameId]
			if game != nil && len(game.Players) < 3 {
				color := map[uint8]string{0: "red", 1: "green", 2: "blue"}[uint8(len(game.Players))]
				game.Players = append(game.Players, model.Player{
					ClientId: clientId,
					Color:    color,
				})

				content := ws.BuildJoinMessage(game)
				// loop through all players and tell them that people has joined
				for _, player := range game.Players {
					server.pool.Clients[player.ClientId].Conn.WriteJSON(content)
				}

				// start game
				if len(game.Players) == 3 {
					go server.updateGameStateForPlayers(game)
				}
			}

		}
	case method == "play":
		clientId, okClientId := message["clientId"].(string)
		gameId, okGameId := message["gameId"].(string)
		cellId, okCellId := message["cellId"].(float64)
		if okClientId && okGameId && okCellId {
			var player *model.Player = nil
			game := server.games[gameId]
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

func (server *Server) updateGameStateForPlayers(game *model.Game) {
	for !game.IsFinished {

		time.Sleep(500 * time.Millisecond)

		content := ws.BuildUpdateMessage(game)
		// loop through all players and send updated state of the game
		for _, player := range game.Players {
			server.pool.Clients[player.ClientId].Conn.WriteJSON(content)
		}
	}
}
