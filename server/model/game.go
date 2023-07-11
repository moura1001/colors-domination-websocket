package model

type Game struct {
	Id         string               `json:"id"`
	Cells      uint8                `json:"cells"`
	Players    map[uint8]*Player    `json:"players"`
	BoardState map[uint8]*CellOwner `json:"state"`
	IsFinished bool                 `json:"-"`
	Winner     *Player              `json:"-"`
}

type Player struct {
	ClientId           string `json:"clientId"`
	Color              string `json:"color"`
	Score              uint8  `json:"-"`
	QueueEntryPosition uint8  `json:"queueId"`
}

type CellOwner struct {
	Color   string `json:"color"`
	OwnerId uint8  `json:"-"`
}
