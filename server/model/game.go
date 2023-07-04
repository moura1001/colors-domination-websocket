package model

type Game struct {
	Id         string           `json:"id"`
	Cells      uint8            `json:"cells"`
	Players    []Player         `json:"players"`
	BoardState map[uint8]string `json:"state"`
}

type Player struct {
	ClientId string `json:"clientId"`
	Color    string `json:"color"`
}
