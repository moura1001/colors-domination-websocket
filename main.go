package main

import (
	"log"
	"net/http"

	server "github.com/moura1001/websocket-colors-domination/server/handler"
)

func main() {
	log.Println("Attempting to start server on port 4000...")

	server.NewServer().SetupRoutes()

	if err := http.ListenAndServe(":4000", nil); err != nil {
		log.Fatalf("could not listen on port 4000: %v", err)
	}
}
