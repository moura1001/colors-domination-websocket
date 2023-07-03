package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/google/uuid"
	ws "github.com/moura1001/websocket-colors-domination/server/websocket"
)

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

func SetupRoutes() {
	fs := http.FileServer(http.Dir("./client/static"))
	http.Handle("/", fs)

	pool := ws.NewPool()
	go pool.Start()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(pool, w, r)
	})
}
