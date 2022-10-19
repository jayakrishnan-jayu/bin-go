package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"github.com/jayakrishnan-jayu/bin-go/bingo"
	"github.com/jayakrishnan-jayu/bin-go/utils"
)


var port = flag.Int("p", 8080, "Port address of the server")


func main() {
	
	ip, err := utils.GetLocalIP()
	if err != nil {
		log.Println(err)
		ip = "localhost"
	}
	addr := fmt.Sprintf("%s:%d", ip, *port)
	bingo := bingo.New(net.ParseIP(ip))

	http.HandleFunc("/", bingo.ServeHttp)
	log.Printf("Starting Server on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
	

	
}