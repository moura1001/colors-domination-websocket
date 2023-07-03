package main

import (
	"log"
	"net/http"

	"github.com/moura1001/websocket-colors-domination/server"
)

func main() {
	log.Println("Attempting to start server on port 4000...")
	server.SetupRoutes()

	if err := http.ListenAndServe(":4000", nil); err != nil {
		log.Fatalf("could not listen on port 4000: %v", err)
	}
}
