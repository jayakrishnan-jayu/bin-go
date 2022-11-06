package bingo

const (
	errorCommand int = iota
	PlayerNameCommand
	PlayersListCommand
	GameConfigCommand
	PlayerBoardCommand
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

type PlayersBoard struct {
	Command int        `json:"command"`
	Board   *[][]uint8 `json:"board"`
}
type GameConfig struct {
	Command     int   `json:"command"`
	IsLobbyMode bool  `json:"is_lobby_mode"`
	BoardSize   uint8 `json:"board_size"`
}

func (g *Game) playerList() PlayersList {
	clients := make([]*Client, 0, len(g.clients))
	for c := range g.clients {
		clients = append(clients, c)
	}
	pList := PlayersList{
		Command: PlayersListCommand,
		Players: clients,
	}
	return pList
}

func (g *Game) gameConfig() GameConfig {
	return GameConfig{
		Command:     GameConfigCommand,
		IsLobbyMode: g.IsLobbyMode,
		BoardSize:   g.BoardSize,
	}
}
