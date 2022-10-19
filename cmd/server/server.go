package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	// "os/signal"
	// "syscall"

	"github.com/gorilla/websocket"
	"github.com/jayakrishnan-jayu/bin-go/utils"
)


var port = flag.Int("p", 8080, "Port address of the server")

var upgrader = websocket.Upgrader{}

func serve(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
		os.Exit(0)
	}
	defer c.Close()

	fmt.Println(r.URL.Host)

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
	
}

func main() {
	
	ip, err := utils.GetLocalIP()
	if err != nil {
		log.Println(err)
		ip = "localhost"
	}
	addr := fmt.Sprintf("%s:%d", ip, *port)
	

	http.HandleFunc("/", serve)
	log.Printf("Starting Server on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
	

	
}