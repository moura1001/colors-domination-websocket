package server

import (
	"net/http"
)

func SetupRoutes() {
	fs := http.FileServer(http.Dir("./client/static"))
	http.Handle("/", fs)

	http.HandleFunc("/ws", wsEndpoint)
}
