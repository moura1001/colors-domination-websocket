package server

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/moura1001/websocket-colors-domination/server/model"
	ws "github.com/moura1001/websocket-colors-domination/server/websocket"
)

type Server struct {
	pool      *ws.Pool
	games     map[string]*model.Game
	stateLock *sync.RWMutex
}

func NewServer() *Server {
	server := new(Server)
	server.stateLock = &sync.RWMutex{}
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
					Players:    map[uint8]*model.Player{},
					BoardState: map[uint8]*model.CellOwner{},
				}

				for i := uint8(0); i < game.Cells; i++ {
					game.BoardState[i] = &model.CellOwner{
						Color:   "",
						OwnerId: 16,
					}
				}

				server.stateLock.Lock()
				server.games[gameId] = game
				server.stateLock.Unlock()

				content := ws.BuildCreateMessage(game)
				clientConn.Conn.WriteJSON(content)
			}

		}
	case method == "join":
		clientId, okClientId := message["clientId"].(string)
		gameId, okGameId := message["gameId"].(string)

		if okClientId && okGameId {
			server.stateLock.RLock()
			game := server.games[gameId]

			if game != nil {
				numberOfPlayers := uint8(len(game.Players))

				if numberOfPlayers < 3 {
					color := map[uint8]string{0: "red", 1: "green", 2: "blue"}[numberOfPlayers]
					server.stateLock.RUnlock()

					server.stateLock.Lock()
					game.Players[numberOfPlayers] = &model.Player{
						ClientId:           clientId,
						Color:              color,
						Score:              uint8(0),
						QueueEntryPosition: numberOfPlayers,
					}
					server.stateLock.Unlock()

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

			} else {
				server.stateLock.RUnlock()
			}

		}
	case method == "play":
		clientId, okClientId := message["clientId"].(float64)
		gameId, okGameId := message["gameId"].(string)
		cellId, okCellId := message["cellId"].(float64)

		if okClientId && okGameId && okCellId {

			server.stateLock.RLock()
			game := server.games[gameId]
			if game != nil {

				player := game.Players[uint8(clientId)]
				server.stateLock.RUnlock()

				if player != nil {
					server.updateGameScore(game, player.QueueEntryPosition, uint8(cellId))
				}
			} else {
				server.stateLock.RUnlock()
			}
		}
	}
}

func (server *Server) updateGameStateForPlayers(game *model.Game) {
	for !game.IsFinished {

		time.Sleep(500 * time.Millisecond)

		content := ws.BuildUpdateMessage(game)
		// loop through all players and send updated state of the game
		for i, player := range game.Players {
			client := server.pool.Clients[player.ClientId]
			if client != nil && client.Conn != nil {
				client.Conn.WriteJSON(content)
			} else {
				// remove disconnected players from the game
				server.stateLock.Lock()
				delete(game.Players, i)
				server.stateLock.Unlock()
			}
		}
	}

	server.endGame(game)

}

func (server *Server) updateGameScore(game *model.Game, playerId uint8, cellId uint8) {

	server.stateLock.RLock()
	players := game.Players
	boardState := game.BoardState

	previousOwnerId := boardState[cellId].OwnerId
	previousOwner := players[previousOwnerId]

	newOwner := players[playerId]
	server.stateLock.RUnlock()

	server.stateLock.Lock()
	defer server.stateLock.Unlock()
	if boardState[cellId].Color != newOwner.Color {

		boardState[cellId].Color = newOwner.Color
		boardState[cellId].OwnerId = playerId
		newOwner.Score++

		if newOwner.Score >= game.Cells {
			game.Winner = newOwner
			game.IsFinished = true
			return
		}

		if previousOwner != nil && previousOwner.Score > 0 {
			previousOwner.Score--
		}
	}
}

func (server *Server) endGame(game *model.Game) {
	server.stateLock.Lock()
	delete(server.games, game.Id)
	server.stateLock.Unlock()
	log.Printf("Game '%s' finished with '%v' as the winner\n", game.Id, *game.Winner)

	content := ws.BuildEndMessage(game)
	// loop through all players and send winner final message of the game
	for _, player := range game.Players {
		client := server.pool.Clients[player.ClientId]
		if client != nil && client.Conn != nil {
			client.Conn.WriteJSON(content)
		}
	}
}
