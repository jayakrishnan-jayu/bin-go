package bingo

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jayakrishnan-jayu/bin-go/utils"
)

type Client struct {
	Id   int             `json:"id"`
	Name string          `json:"name"`
	Ip   net.IP          `json:"ip"`
	Conn *websocket.Conn `json:"-"`
	game *Game           `json:"-"`
	Send chan []byte     `json:"-"`
}

func (client *Client) String() string {
	return fmt.Sprintf("Id: %d, Name: %s, Ip: %s", client.Id, client.Name, client.Ip)
}

func (c *Client) SetSocketReadConfig() {
	c.Conn.SetReadLimit(utils.MaxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(utils.PongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(utils.PongWait))
		return nil
	})
}

func (c *Client) ReadMessage() ([]byte, bool) {
	_, message, err := c.Conn.ReadMessage()
	if err != nil {
		if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			log.Printf("error: %v", err)
		} else {
			log.Printf("closed: %v", err)
		}
		return nil, false
	}
	return message, true
}

func (c *Client) HandleSocketPing() error {
	c.Conn.SetWriteDeadline(time.Now().Add(utils.WriteWait))
	if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
		return err
	}
	return nil
}

func (c *Client) writePump() {
	fmt.Println("write pumb")
	ticker := time.NewTicker(utils.PingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(utils.WriteWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}
			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.HandleSocketPing(); err != nil {
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
	c.SetSocketReadConfig()
	for {
		var messageMap map[string]interface{}
		message, ok := c.ReadMessage()
		if !ok {
			break
		}
		err := json.Unmarshal(message, &messageMap)
		if err != nil {
			log.Printf("json: %v", err)
			break
		}
		cmd, ok := utils.GetCommandFromMap(messageMap)
		if !ok {
			break
		}
		c.handlePlayerResponse(cmd, message)
	}
}

func (c *Client) handlePlayerResponse(cmd int, message []byte) {
	switch cmd {
	case PlayerNameCommand:
		var playerUserName PlayerName
		err := json.Unmarshal(message, &playerUserName)
		if err != nil {
			log.Println(err)
			break
		}
		c.Name = playerUserName.Name
		c.game.broadcastPlayerlist()
	}
}
