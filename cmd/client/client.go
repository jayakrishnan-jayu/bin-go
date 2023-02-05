package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jayakrishnan-jayu/bin-go/bingo"
	"github.com/jayakrishnan-jayu/bin-go/utils"
)

var serverIp = flag.String("i", "localhost", "Ip Address of Server")
var port = flag.Int("p", 8080, "Port address of the server")
var username = flag.String("u", "user", "Username for game session")

type Client bingo.Client
type GameConfig bingo.GameConfig

type Game struct {
	gameConfig GameConfig
	board      *[][]uint8
	started    bool
}

var game Game
var playersList bingo.PlayersList
var finished bool

func (c *Client) writePump() {
	ticker := time.NewTicker(utils.PingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			// fmt.Println("Sending", string(message))
			c.Conn.SetWriteDeadline(time.Now().Add(utils.WriteWait))
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
				w.Write(utils.Newline)
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(utils.WriteWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(utils.MaxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(utils.PongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(utils.PongWait))
		return nil
	})
	for {
		messages, ok := c.ReadMessages()
		if !ok {
			break
		}
		for _, message := range messages {
			var messageMap map[string]interface{}
			if err := json.Unmarshal(message, &messageMap); err != nil {
				log.Fatal("mesage map readPumb: ", err)
				return
			}

			val, err := parseCommandMessage(messageMap)
			if err != nil {
				log.Fatal("parse command message readPumb: ", err)
				return
			}
			c.handleServerCommand(val, message)
		}
	}
}

func (c *Client) ReadMessages() ([][]byte, bool) {
	_, message, err := c.Conn.ReadMessage()
	if err != nil {
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			// log.Printf("error: %v", err)
		} else {
			// log.Printf("closed: %v", err)
		}
		return nil, false
	}
	messages := bytes.Split(message, utils.Newline)
	return messages, true
}

func parseCommandMessage(messageMap map[string]interface{}) (int, error) {

	if cmd, ok := messageMap["command"].(float64); ok {
		return int(cmd), nil
	}
	// return -1, fmt.Errorf("Invalid command")

	return -1, fmt.Errorf("Command not found")

}

func (g *Game) generateGameBoard() {
	rand.Seed(time.Now().UnixNano())
	addedNumbers := map[uint8]bool{}
	size := int(g.gameConfig.BoardSize)
	board := make([][]uint8, size)
	for i := range board {
		board[i] = make([]uint8, size)
		var n uint8
		for j := range board[i] {
			for {
				n = uint8(rand.Intn(size*size) + 1)
				if addedNumbers[n] {
					continue
				}
				break
			}
			addedNumbers[n] = true
			board[i][j] = n
		}
	}
	g.board = &board
}

func (c *Client) handleServerCommand(cmd int, message []byte) {
	switch cmd {
	case bingo.PlayerNameCommand:
		output, err := json.Marshal(bingo.PlayerName{
			Command: bingo.PlayerNameCommand,
			Name:    *username,
		})
		if err != nil {
			log.Fatal("handleServerCommand ", err)
			break
		}

		c.Send <- output
	case bingo.PlayerIDCommand:
		var playerId bingo.PlayerID
		err := json.Unmarshal(message, &playerId)
		if err != nil {
			log.Fatal("handleServerCommand ", err)
			break
		}
		c.Id = playerId.ID
	case bingo.PlayersListCommand:
		err := json.Unmarshal(message, &playersList)
		if err != nil {
			log.Fatal("handleServerCommand ", err)
			break
		}
		playersList.RenderLobby()
	case bingo.GameConfigCommand:
		err := json.Unmarshal(message, &game.gameConfig)
		if err != nil {
			log.Fatal("handleServerCommand ", err)
			break
		}
	case bingo.PlayerBoardCommand:
		if game.gameConfig == (GameConfig{}) {
			log.Fatal("handleServerCommand: GameConfig not yet intilzied")
		}
		fmt.Println("Board size", game.gameConfig.BoardSize)
		game.generateGameBoard()
		fmt.Println("Board size", len(*game.board))
		fmt.Printf("%v", game.board)
		output, err := json.Marshal(bingo.PlayersBoard{
			Command: bingo.PlayerBoardCommand,
			Board:   game.board,
		})
		if err != nil {
			log.Fatal("handleServerCommand ", err)
			break
		}

		c.Send <- output
	case bingo.GameStatusCommand:
		if finished {
			break
		}
		bingo.ClearTerminal()
		var gameStatus bingo.GameStatus
		err := json.Unmarshal(message, &gameStatus)
		if err != nil {
			log.Fatal("handleServerCommand ", err)
			break
		}
		bingo.RenderBoard(*game.board)
		fmt.Println("Current Player: ", gameStatus.PlayerId)
		if gameStatus.PlayerId == c.Id {
			go func() {
				fmt.Println("Enter Input ")
				var digit int
				fmt.Scanf("%d", &digit)
				output, err := json.Marshal(bingo.GameMove{
					Command: bingo.GameMoveCommand,
					Change:  uint8(digit),
				})
				if err != nil {
					log.Fatal("handleServerCommand ", err)
				}
				c.Send <- output
			}()
		}
	case bingo.GameMoveCommand:
		var gameMove bingo.GameMove
		err := json.Unmarshal(message, &gameMove)
		if err != nil {
			log.Fatal("handleServerCommand ", err)
			break
		}

		fmt.Println("Current Player: ", gameMove.Change)
	case bingo.GameScoreIndexCommand:
		var scoreIndex bingo.GameScoreIndex
		err := json.Unmarshal(message, &scoreIndex)
		if err != nil {
			log.Fatal("handleServerCommand ", err)
			break
		}
		bingo.ClearTerminal()
		finished = true
		fmt.Printf("You won %d/%d", scoreIndex.Score, len(playersList.Players))
	}
}

func main() {
	flag.Parse()

	addr := fmt.Sprintf("%s:%d", *serverIp, *port)
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: addr, Path: "/ws"}
	// log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}

	client := &Client{
		Name: *username,
		Conn: c,
		Send: make(chan []byte, 256),
	}

	done := make(chan struct{})

	go func() {
		defer close(done)
		client.writePump()
	}()
	go func() {
		defer close(done)
		client.readPump()
	}()

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("interrupt")
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}

}
