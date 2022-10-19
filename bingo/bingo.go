package bingo

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

type Client struct {
	Id int `json:"id"`
	Name string `json:"name"`
	Ip net.IP `json:"ip"`
	Conn *websocket.Conn `json:"-"`
}

func (client *Client) String() string {
	return fmt.Sprintf("Id: %d, Name: %s, Ip: %s", client.Id, client.Name, client.Ip)
}

type Game struct {
	players[] Client
	IsLobbyMode bool
	playerIndex int
}

func (game *Game) String() string {
	return fmt.Sprintf("Clients: %v", game.players)
}


func (game *Game) handleNewPlayer(con *websocket.Conn) {
	if ipnet, ok := con.LocalAddr().(*net.TCPAddr); ok {
		game.playerIndex += 1
		game.players = append(
			game.players, 
			Client{
				Ip: ipnet.IP,
				Id: game.playerIndex,
				Conn: con,
			},
		)
	}
}

func (game *Game) handlePlayerExit(con *websocket.Conn) {
	for i, player := range game.players {
		if con == player.Conn {
			game.players[i] = game.players[len(game.players)-1]
    		game.players = game.players[:len(game.players)-1]
			return;
		}
	}
}

func (game *Game) broadcastPlayerInfo() {
	for _, player := range game.players {
		if player.Id == 0 {
			continue
		}
		jsonData, err := json.Marshal(&game.players)
		fmt.Println(jsonData)
		if err != nil {
			log.Println("write:", err)
			break
		}
		err = player.Conn.WriteMessage(websocket.TextMessage, []byte(string(jsonData)))
		if err != nil {
			log.Println("write:", err)
			break
		}
		
	}
}

func (game *Game) ServeHttp(w http.ResponseWriter, r *http.Request) {

	if !game.IsLobbyMode {
		http.Error(w, "This Server is not accepting anymore players", http.StatusForbidden)
		return
	}

	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
		os.Exit(0)
	}
	defer c.Close()


	game.handleNewPlayer(c)
	game.broadcastPlayerInfo()
	
	fmt.Println(game)

	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)
		err = c.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
	game.handlePlayerExit(c)
	game.broadcastPlayerInfo()
	fmt.Println(game)
}

func New(serverIp net.IP) *Game {
	rootClient := &Client{
		Ip: serverIp,
		Name: "Server",
		Id: 0,
	}
	game := &Game{
		players: []Client{*rootClient},
		IsLobbyMode: true,
	}
	return game
}





