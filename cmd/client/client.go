package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
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
type Game bingo.Game

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
			fmt.Println(message)
			if err := json.Unmarshal(message, &messageMap); err != nil {
				log.Fatal("mesage map readPumb: ", err)
				return
			}
			// fmt.Printf("Server: %s\n", message)

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

	case bingo.PlayersListCommand:
		var playersList bingo.PlayersList
		err := json.Unmarshal(message, &playersList)
		if err != nil {
			log.Fatal("handleServerCommand ", err)
			break
		}
		playersList.RenderLobby()
	case bingo.GameConfigCommand:
		var gc bingo.GameConfig
		err := json.Unmarshal(message, &gc)
		if err != nil {
			log.Fatal("handleServerCommand ", err)
			break
		}
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
