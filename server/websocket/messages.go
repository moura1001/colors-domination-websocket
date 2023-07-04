package websocket

import "github.com/moura1001/websocket-colors-domination/server/model"

type Message map[string]interface{}

func BuildConnectMessage(clientId string) Message {
	return Message{
		"method":   "connect",
		"clientId": clientId,
	}
}

func BuildCreateMessage(game *model.Game) Message {

	return Message{
		"method": "create",
		"game":   game,
	}
}
