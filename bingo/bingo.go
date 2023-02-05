package bingo

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

type Game struct {
	IsLobbyMode bool
	BoardSize   uint8
	playerIndex uint8
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Inbound messages from the clients.
	receive chan GameMove

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	// Input from Host
	input chan string

	// Score Index to print on the scoreboard
	scoreIndex uint8

	// Board values: true exists, false does not exist
	values *[][]bool
}

func (game *Game) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// fmt.Println("new Connection")
	if !game.IsLobbyMode {
		http.Error(w, "This Server is not accepting anymore players", http.StatusForbidden)
		return
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal("ServeHTTP: ", err)
	}

	ipnet, ok := c.LocalAddr().(*net.TCPAddr)
	if !ok {
		log.Fatal("ServeHTTP: Could not find IP")
	}
	game.playerIndex += 1
	client := &Client{
		Id:   game.playerIndex,
		Ip:   ipnet.IP,
		Conn: c,
		game: game,
		Send: make(chan []byte, 256),
		score: 0,
		scoreIndex: 0,
	}
	// fmt.Println("new user")

	game.register <- client

	go client.writePump()
	go client.readPump()

	client.requestPlayerName()
	client.sendPlayerID()
	client.sendGameConfig()
	client.requestGeneratedBoard()
}

func (g *Game) broadcastPlayerlist() {
	output, err := json.Marshal(g.playerList())
	if err != nil {
		log.Fatal("broadcastPlayerlist: ", err)
		return
	}
	g.broadcast <- output
}

func (g *Game) sendGameStatus(playerId uint8) {
	cmd := GameStatus{
		Command:  GameStatusCommand,
		PlayerId: playerId,
	}
	output, err := json.Marshal(cmd)
	if err != nil {
		log.Fatal("requestClientName: ", err)
	}
	g.broadcast <- output
}

func (c *Client) requestPlayerName() {
	cmd := RequestCommand{Command: PlayerNameCommand}
	output, err := json.Marshal(cmd)
	if err != nil {
		log.Fatal("requestClientName: ", err)
	}
	c.Send <- output
}

func (c *Client) sendPlayerID() {
	cmd := PlayerID{
		Command: PlayerIDCommand,
		ID:      c.Id,
	}
	output, err := json.Marshal(cmd)
	if err != nil {
		log.Fatal("requestGenerateBoard: ", err)
	}
	c.Send <- output
}

func (c *Client) requestGeneratedBoard() {
	cmd := RequestCommand{Command: PlayerBoardCommand}
	output, err := json.Marshal(cmd)
	if err != nil {
		log.Fatal("requestGenerateBoard: ", err)
	}
	c.Send <- output
}

func (c *Client) sendGameConfig() {
	output, err := json.Marshal(c.game.gameConfig())
	if err != nil {
		log.Fatal("sendGameConfig:", err)
		return
	}
	c.Send <- output
}

func (c *Client) sendGameScoreIndex() {
	cmd := GameScoreIndex{Command: GameScoreIndexCommand, Score: c.scoreIndex}
	output, err := json.Marshal(cmd)
	if err != nil {
		log.Fatal("requestGenerateBoard: ", err)
	}
	c.Send <- output

}

func New(serverIp net.IP) *Game {
	game := &Game{
		IsLobbyMode: true,
		BoardSize:   5,
		broadcast:   make(chan []byte),
		receive:     make(chan GameMove),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		clients:     make(map[*Client]bool),
		input:       make(chan string),
		scoreIndex: 1,
	}
	
	values := make([][]bool, game.BoardSize)
	for i := range values {
		values[i] = make([]bool, game.BoardSize)
        for j := range values[i] {
            values[i][j] = true
        }
	}
	game.values = &values
	return game
}
func (g *Game) readInput() {

	for {
		var u string
		fmt.Scanf("%s\n", &u)
		g.input <- u
	}
}

func (g *Game) updateTable(n uint8) {
	n -= 1
	i := n / g.BoardSize
	j := n % g.BoardSize
	(*g.values)[i][j] = false 
}

func (g *Game) isCrossed(n uint8) bool {
	n -= 1
	i := n / g.BoardSize
	j := n % g.BoardSize
	return !(*g.values)[i][j]
}

func (g *Game) computePlayerScore(board *[][]uint8) (rows, cols, diags uint8){
	n := len(*board)
    diagCrossed := true
    inverseDiagCrossed := true
    for i := 0; i < n; i++ {
        rowCrossed := true
        colCrossed := true
        for j := 0; j < n; j++ {
            rowCrossed = rowCrossed && g.isCrossed((*board)[i][j])
            colCrossed = colCrossed && g.isCrossed((*board)[j][i])
            if i == j {
                diagCrossed = diagCrossed && g.isCrossed((*board)[i][j])
                inverseDiagCrossed = inverseDiagCrossed && g.isCrossed((*board)[i][n-1-i])
            }
        }
        if rowCrossed {
            rows++
        }
        if colCrossed {
            cols++
        }
    }
	if diagCrossed {
		diags++
	}
	if inverseDiagCrossed {
		diags++
	}
    return
}

func (g* Game) renderScoreBoard() {
	scoreIndexChanged := false
	fmt.Println("\n\nStart")
	for c := range g.clients {
		fmt.Printf("%s - %d\n", c.Name, c.score)
		if c.score < g.BoardSize {
			
			row, col, diag := g.computePlayerScore(c.board)
			
			c.score = row+col+diag
			fmt.Printf("New Score %d\n", c.score)
			if c.score >= g.BoardSize {
				scoreIndexChanged = true
				c.scoreIndex = g.scoreIndex
				fmt.Printf("Score Index %d\n", c.scoreIndex)
				go c.sendGameScoreIndex()
			}
		}
		
	}
	if scoreIndexChanged {
		g.scoreIndex +=1
		scoreIndexChanged = false
	}
	// RenderServerBoard(&g.clients)
}

func (g *Game) play() {
	ClearTerminal()
	clients := make([]*Client, 0, len(g.clients))
	for c := range g.clients {
		clients = append(clients, c)
	}
	for {
		for _, c := range clients {
			if _, ok := g.clients[c]; ok && c.scoreIndex <= 0{
				g.sendGameStatus(c.Id)
				gameMove := <-g.receive
				if gameMove.Author != c {
					log.Fatal("play: gameMove author assertion failed")
				}
				g.updateTable(gameMove.Change)
				g.renderScoreBoard()
				fmt.Printf("%s update: %d\n", gameMove.Author.Name, gameMove.Change)

			}
		}
	}
}

func (g *Game) Run() {
	go g.readInput()
	for {
		select {
		case client := <-g.register:
			fmt.Println("got user to regiser", client)
			g.clients[client] = true
			g.playerList().RenderLobby()
			fmt.Println("Enter s to start game")
			// fmt.Println("regiseterd new user")
		case client := <-g.unregister:
			for {
				if _, ok := g.clients[client]; ok {
					close(client.Send)
					client.Conn.Close()
					delete(g.clients, client)
				}
				if len(g.unregister) > 0 {
					client = <-g.unregister
					continue
				}
				if len(g.clients) > 0 {
					go g.broadcastPlayerlist()
				}
				break
			}
			g.playerList().RenderLobby()
			fmt.Println("Enter s to start game")

		// case message := <-g.receive:
		// fmt.Println(message)

		case message := <-g.broadcast:
			for client := range g.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(g.clients, client)
				}
			}
		case cmd := <-g.input:
			switch cmd {
			case "s":
				if g.IsLobbyMode {
					g.IsLobbyMode = false
					go g.play()
				}
			}
		}
	}
}
