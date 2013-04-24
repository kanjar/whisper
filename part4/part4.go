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

type Message struct {
	Body string
}

func main() {
	go listen()

	peers, err := util.ListPeers()
	if err != nil {
		log.Fatal(err)
	}
	for _, peer := range peers {
		go connect(peer)
	}

	r := bufio.NewReader(os.Stdin)
	for {
		s, err := r.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		m := Message{Body: s}
		for _, peerc := range Peers {
			peerc <- m
		}
	}
}

var Peers = make(map[string]chan Message)

func connect(peer string) {
	c, err := net.Dial("tcp", peer)
	if err != nil {
		log.Println(err)
		return
	}

	msgCh := make(chan Message)
	Peers[peer] = msgCh

	e := json.NewEncoder(c)
	for m := range msgCh {
		err = e.Encode(m)
		if err != nil {
			log.Println(err)
			return
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
