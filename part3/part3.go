package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/nf/whisper/util"
)

const remoteAddr = "localhost:8000"

type Message struct {
	Body string
}

func main() {
	go listen()

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

func listen() {
	l, err := util.Listen()
	if err != nil {
		log.Fatal(err)
	}

	err = util.RegisterPeer(l.Addr().String())
	if err != nil {
		log.Fatal(err)
	}

	for {
		c, err := l.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go handle(c)
	}
}

func handle(c net.Conn) {
	d := json.NewDecoder(c)
	for {
		var m Message
		err := d.Decode(&m)
		if err != nil {
			log.Println(err)
			continue
		}
		fmt.Println(m.Body)
	}
}
