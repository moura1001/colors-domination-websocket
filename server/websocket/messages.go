package websocket

type Message map[string]interface{}

func BuildConnectMessage(clientId string) Message {
	return Message{
		"method":   "connect",
		"clientId": clientId,
	}
}
