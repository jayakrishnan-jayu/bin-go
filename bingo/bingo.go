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
	receive chan map[string]interface{}

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	// Input from Host
	input chan string
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
	}
	// fmt.Println("new user")

	game.register <- client

	go client.writePump()
	go client.readPump()

	client.requestClientName()
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

func (c *Client) requestClientName() {
	cmd := RequestCommand{Command: PlayerNameCommand}
	output, err := json.Marshal(cmd)
	if err != nil {
		log.Fatal("requestClientName: ", err)
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

func New(serverIp net.IP) *Game {
	game := &Game{
		IsLobbyMode: true,
		BoardSize:   5,
		broadcast:   make(chan []byte),
		receive:     make(chan map[string]interface{}),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		clients:     make(map[*Client]bool),
		input:       make(chan string),
	}
	return game
}
func (g *Game) readInput() {

	for {
		var u string
		fmt.Scanf("%s\n", &u)
		g.input <- u
	}
}

func (g *Game) Run() {
	go g.readInput()
	for {
		select {
		case client := <-g.register:
			// fmt.Println("got user to regiser", client)
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
				g.IsLobbyMode = !g.IsLobbyMode
				fmt.Println("Lobby mode: ", g.IsLobbyMode)
			}
		}
	}
}
