package bingo

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 10 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 8) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{}

type Client struct {
	Id   int             `json:"id"`
	Name string          `json:"name"`
	Ip   net.IP          `json:"ip"`
	Conn *websocket.Conn `json:"-"`
	game *Game
	Send chan []byte
}

func (client *Client) String() string {
	return fmt.Sprintf("Id: %d, Name: %s, Ip: %s", client.Id, client.Name, client.Ip)
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.game.unregister <- c
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			} else {
				log.Printf("closed: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		c.game.broadcast <- message
	}
}

type Game struct {
	players     []Client
	IsLobbyMode bool
	playerIndex int
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func (game *Game) String() string {
	return fmt.Sprintf("Clients: %v", game.players)
}

func (game *Game) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
		Name: fmt.Sprintf("Player_%d", game.playerIndex),
		Ip:   ipnet.IP,
		Conn: c,
		game: game,
		Send: make(chan []byte, 256),
	}

	game.register <- client
	fmt.Println(client.String())

	go client.writePump()
	go client.readPump()
}

// func (c *Client) handlePlayerDelete() {
// }

func New(serverIp net.IP) *Game {
	game := &Game{
		IsLobbyMode: true,
		broadcast:   make(chan []byte),
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
			g.clients[client] = true
		case client := <-g.unregister:
			if _, ok := g.clients[client]; ok {
				close(client.Send)
				delete(g.clients, client)
			}
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
