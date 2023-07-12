package websocket

import "github.com/moura1001/websocket-colors-domination/server/model"

type Message map[string]interface{}

type OnReceivedMessageHandler func(method string, message Message)

func BuildConnectMessage(clientId string) Message {
	return Message{
		"method":   "connect",
		"clientId": clientId,
	}
}

func BuildCreateMessage(game *model.Game) Message {
	return buildGameMessage("create", game)
}

func BuildJoinMessage(game *model.Game) Message {
	return buildGameMessage("join", game)
}

func BuildUpdateMessage(game *model.Game) Message {
	return buildGameMessage("update", game)
}

func buildGameMessage(method string, game *model.Game) Message {
	return Message{
		"method": method,
		"game":   game,
	}
}

func BuildEndMessage(game *model.Game) Message {
	if game.Winner != nil {
		return Message{
			"method": "end",
			"winner": *game.Winner,
		}
	} else {
		return Message{
			"method": "end",
		}
	}
}

func BuildCPUMessage() Message {
	return Message{
		"method": "cpu",
	}
}
