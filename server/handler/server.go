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

type gameInfo struct {
	game      *model.Game
	stateLock *sync.RWMutex
}

type Server struct {
	pool              *ws.Pool
	games             map[string]*gameInfo
	gamesLock         *sync.RWMutex
	maxPlayersPerGame uint8
}

func NewServer() *Server {
	server := new(Server)
	server.maxPlayersPerGame = 3
	server.gamesLock = &sync.RWMutex{}
	server.games = map[string]*gameInfo{}
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

	pool.Register(client)
	client.Read()
}

func (server *Server) SetupRoutes() {
	fs := http.FileServer(http.Dir("./client/static"))
	http.Handle("/", fs)

	go server.pool.HandleBroadcast()

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

				gameInfo := &gameInfo{
					game:      game,
					stateLock: &sync.RWMutex{},
				}

				server.gamesLock.Lock()
				server.games[gameId] = gameInfo
				server.gamesLock.Unlock()

				content := ws.BuildCreateMessage(game)
				clientConn.Conn.WriteJSON(content)
			}

		}
	case method == "join":
		clientId, okClientId := message["clientId"].(string)
		gameId, okGameId := message["gameId"].(string)

		if okClientId && okGameId {
			server.gamesLock.RLock()
			gameInfo := server.games[gameId]

			if gameInfo != nil {
				server.gamesLock.RUnlock()

				gameInfo.stateLock.RLock()

				if uint8(len(gameInfo.game.Players)) < server.maxPlayersPerGame {

					availableQueueId := uint8(0)
					for i := uint8(0); i < server.maxPlayersPerGame; i++ {
						_, exist := gameInfo.game.Players[i]
						if !exist {
							availableQueueId = i
							break
						}
					}

					color := map[uint8]string{0: "red", 1: "green", 2: "blue"}[availableQueueId]
					gameInfo.stateLock.RUnlock()

					gameInfo.stateLock.Lock()
					gameInfo.game.Players[availableQueueId] = &model.Player{
						ClientId: clientId,
						Color:    color,
						Score:    uint8(0),
						QueueId:  availableQueueId,
					}
					gameInfo.stateLock.Unlock()

					content := ws.BuildJoinMessage(gameInfo.game)
					// loop through all players and tell them that people has joined
					for _, player := range gameInfo.game.Players {
						server.pool.Clients[player.ClientId].Conn.WriteJSON(content)
					}

					// start game
					gameInfo.stateLock.RLock()
					if uint8(len(gameInfo.game.Players)) == server.maxPlayersPerGame && !gameInfo.game.IsStarted {
						gameInfo.stateLock.RUnlock()

						gameInfo.stateLock.Lock()
						gameInfo.game.IsStarted = true
						gameInfo.stateLock.Unlock()

						go server.updateGameStateForPlayers(gameInfo)
					} else {
						gameInfo.stateLock.RUnlock()
					}

				} else {
					gameInfo.stateLock.RUnlock()
				}

			} else {
				server.gamesLock.RUnlock()
			}

		}
	case method == "play":
		clientId, okClientId := message["clientId"].(float64)
		gameId, okGameId := message["gameId"].(string)
		cellId, okCellId := message["cellId"].(float64)

		if okClientId && okGameId && okCellId {

			server.gamesLock.RLock()
			gameInfo := server.games[gameId]
			if gameInfo != nil {
				server.gamesLock.RUnlock()

				gameInfo.stateLock.RLock()
				player := gameInfo.game.Players[uint8(clientId)]
				gameInfo.stateLock.RUnlock()

				if player != nil {
					server.updateGameScore(gameInfo, player.QueueId, uint8(cellId))
				}
			} else {
				server.gamesLock.RUnlock()
			}
		}
	case method == "cpu":
		clientId, okClientId := message["clientId"].(float64)
		gameId, okGameId := message["gameId"].(string)

		if okClientId && okGameId {
			server.gamesLock.RLock()
			gameInfo := server.games[gameId]

			if gameInfo != nil {
				server.gamesLock.RUnlock()

				gameInfo.stateLock.RLock()
				player := gameInfo.game.Players[uint8(clientId)]
				gameInfo.stateLock.RUnlock()

				if player != nil {
					server.cpuMode(gameInfo.game)
				}

			} else {
				server.gamesLock.RUnlock()
			}

		}
	}
}

func (server *Server) updateGameStateForPlayers(gameInfo *gameInfo) {
	log.Printf("Starting updates for the game '%s'\n", gameInfo.game.Id)

	for !gameInfo.game.IsFinished {

		time.Sleep(500 * time.Millisecond)

		content := ws.BuildUpdateMessage(gameInfo.game)
		// loop through all players and send updated state of the game
		for i, player := range gameInfo.game.Players {
			client := server.pool.Clients[player.ClientId]
			if client != nil && client.Conn != nil {
				client.Conn.WriteJSON(content)
			} else {
				// remove disconnected players from the game
				gameInfo.stateLock.Lock()
				delete(gameInfo.game.Players, i)
				gameInfo.stateLock.Unlock()
			}
		}
	}

	server.endGame(gameInfo.game)

}

func (server *Server) updateGameScore(gameInfo *gameInfo, playerId uint8, cellId uint8) {

	gameInfo.stateLock.RLock()
	players := gameInfo.game.Players
	boardState := gameInfo.game.BoardState

	previousOwnerId := boardState[cellId].OwnerId
	previousOwner := players[previousOwnerId]

	newOwner := players[playerId]
	gameInfo.stateLock.RUnlock()

	gameInfo.stateLock.Lock()
	defer gameInfo.stateLock.Unlock()
	if boardState[cellId].Color != newOwner.Color {

		boardState[cellId].Color = newOwner.Color
		boardState[cellId].OwnerId = playerId
		newOwner.Score++

		if newOwner.Score >= gameInfo.game.Cells {
			gameInfo.game.Winner = newOwner
			gameInfo.game.IsFinished = true
			return
		}

		if previousOwner != nil && previousOwner.Score > 0 {
			previousOwner.Score--
		}
	}
}

func (server *Server) endGame(game *model.Game) {
	server.gamesLock.Lock()
	delete(server.games, game.Id)
	server.gamesLock.Unlock()
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

func (server *Server) cpuMode(game *model.Game) {
	log.Printf("Game '%s' started on cpu battle mode\n", game.Id)

	content := ws.BuildCPUMessage()
	// loop through all players and set cpu battle mode
	for _, player := range game.Players {
		client := server.pool.Clients[player.ClientId]
		if client != nil && client.Conn != nil {
			client.Conn.WriteJSON(content)
		}
	}

}
