package websocket

import (
	"log"
	"sync"
)

type Pool struct {
	Clients         map[string]*Client
	Broadcast       chan Message
	messageHandler  OnReceivedMessageHandler
	connectionsLock *sync.RWMutex
}

func NewPool(messageHandler OnReceivedMessageHandler) *Pool {
	return &Pool{
		Clients:         map[string]*Client{},
		Broadcast:       make(chan Message, 1000),
		messageHandler:  messageHandler,
		connectionsLock: &sync.RWMutex{},
	}
}

func (pool *Pool) Register(client *Client) {
	if client.ID != "" {
		pool.connectionsLock.Lock()
		pool.Clients[client.ID] = client
		pool.connectionsLock.Unlock()

		log.Printf("client '%s' connected", client.ID)
		log.Println("Size of Connection Pool has been increased to: ", len(pool.Clients))

		content := BuildConnectMessage(client.ID)
		client.Conn.WriteJSON(content)
	}
}

func (pool *Pool) Unregister(client *Client) {

	pool.connectionsLock.RLock()
	_, exist := pool.Clients[client.ID]
	if exist {
		pool.connectionsLock.RUnlock()

		pool.connectionsLock.Lock()
		delete(pool.Clients, client.ID)
		pool.connectionsLock.Unlock()

		log.Printf("client '%s' disconnected", client.ID)
		log.Println("Size of Connection Pool has been decreased to: ", len(pool.Clients))
	} else {
		pool.connectionsLock.RUnlock()
	}

}

func (pool *Pool) HandleBroadcast() {
	for message := range pool.Broadcast {
		log.Println("Received message: ", message)

		_, exist := message["method"]

		if exist {

			method, ok := message["method"].(string)
			if ok {
				go pool.messageHandler(method, message)
			}
		}
	}
}
