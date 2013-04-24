package main

import (
	"bufio"
	"encoding/json"
	"log"
	"net"
	"os"
)

const remoteAddr = "localhost:8000"

type Message struct {
	Body string
}

func main() {
	c, err := net.Dial("tcp", remoteAddr)
	if err != nil {
		log.Fatal(err)
	}
	r := bufio.NewReader(os.Stdin)
	e := json.NewEncoder(c)
	for {
		s, err := r.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		m := Message{Body: s}
		err = e.Encode(m)
		if err != nil {
			log.Fatal(err)
		}
	}
}
