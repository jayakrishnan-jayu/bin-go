package main

import (
	// "encoding/json"
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
)

var serverIp = flag.String("i", "localhost", "Ip Address of Server")
var port = flag.Int("p", 8080, "Port address of the server")


func main() {
	flag.Parse()

	addr := fmt.Sprintf("%s:%d", *serverIp, *port)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: addr, Path: "/echo"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
					log.Println("readMessage()", err)
					break
				}
			var client[] bingo.Client
				
			err = json.Unmarshal(message, &client)
			
			if err != nil {
				log.Println("readjson()", err)
				break
			}
			fmt.Println("Client:", client)
		}
	}()


	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
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