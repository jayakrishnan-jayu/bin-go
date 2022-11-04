package bingo

const (
	errorCommand int = iota
	PlayerNameCommand
	PlayersListCommand
)

type RequestCommand struct {
	Command int `json:"command"`
}

type PlayerName struct {
	Command int    `json:"command"`
	Name    string `json:"name"`
}

type PlayersList struct {
	Command int       `json:"command"`
	Players []*Client `json:"players"`
}
