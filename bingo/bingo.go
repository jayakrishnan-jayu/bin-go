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
	playerIndex int
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
}

func (game *Game) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("new Connection")
	if !game.IsLobbyMode {
		http.Error(w, "This Server is not accepting anymore players", http.StatusForbidden)
		return
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}

	ipnet, ok := c.LocalAddr().(*net.TCPAddr)
	if !ok {
		log.Fatal(fmt.Errorf("Could not find IP"))
	}
	game.playerIndex += 1
	client := &Client{
		Id:   game.playerIndex,
		Ip:   ipnet.IP,
		Conn: c,
		game: game,
		Send: make(chan []byte, 256),
	}
	fmt.Println("new user")

	game.register <- client

	go client.writePump()
	go client.readPump()

	cmd := RequestCommand{Command: PlayerNameCommand}
	output, err := json.Marshal(cmd)
	if err != nil {
		log.Println(err)
	}
	fmt.Println("Sending Command", string(output))
	client.Send <- output
}

func (g *Game) broadcastPlayerlist() {
	clients := make([]*Client, 0, len(g.clients))
	for c2 := range g.clients {
		clients = append(clients, c2)
	}
	pList := PlayersList{
		Command: PlayersListCommand,
		Players: clients,
	}

	output, err := json.Marshal(pList)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Println("Sending", string(output))
	g.broadcast <- output
}

func New(serverIp net.IP) *Game {
	game := &Game{
		IsLobbyMode: true,
		broadcast:   make(chan []byte),
		receive:     make(chan map[string]interface{}),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		clients:     make(map[*Client]bool),
	}
	return game
}

func (g *Game) Run() {
	for {
		select {
		case client := <-g.register:
			fmt.Println("got user to regiser", client)
			g.clients[client] = true
			fmt.Println("regiseterd new user")
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
		case message := <-g.receive:
			fmt.Println(message)

		case message := <-g.broadcast:
			for client := range g.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(g.clients, client)
				}
			}
		}
	}
}
