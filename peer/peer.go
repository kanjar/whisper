package main

import (
	"bufio"
	"crypto/rand"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/nf/whisper/pkg/helper"
	"github.com/nf/whisper/pkg/master"
)

const (
	refreshInterval = 5 * time.Second
	sendTimeout     = 5 * time.Second
	defaultTTL      = 5
)

type Message struct {
	ID   string
	Body string
	TTL  int
}

var Peers = struct {
	sync.RWMutex
	m map[string]chan<- Message
}{m: make(map[string]chan<- Message)}

var Messages = struct {
	sync.Mutex
	m map[string]bool
}{m: make(map[string]bool)}

func main() {
	flag.Parse()

	l, err := helper.Listen()
	if err != nil {
		log.Fatal(err)
	}
	go accept(l)

	go readInput()

	self := l.Addr().String()
	err = master.RegisterPeer(self)
	if err != nil {
		log.Fatal(err)
	}
	for {
		addrs, err := master.ListPeers()
		if err != nil {
			log.Println(err)
			continue
		}
		for _, addr := range addrs {
			if addr != self {
				go connect(addr)
			}
		}
		time.Sleep(refreshInterval)
	}
}

func readInput() {
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		id := randomID()
		Messages.Lock()
		Messages.m[id] = true
		Messages.Unlock()
		broadcast(Message{
			ID:   id,
			Body: s.Text(),
			TTL:  defaultTTL,
		})
	}
	log.Fatal(s.Err())
}

func randomID() string {
	b := make([]byte, 20)
	n, _ := rand.Read(b)
	return fmt.Sprintf("%x", b[:n])
}

func connect(addr string) {
	// Don't connect if we're already connected.
	Peers.RLock()
	_, ok := Peers.m[addr]
	Peers.RUnlock()
	if ok {
		return
	}

	// Connect to the peer.
	c, err := net.Dial("tcp", addr)
	if err != nil {
		log.Println(err)
		return
	}
	defer c.Close()

	log.Println("connected to", addr)
	defer log.Println("disconnected from", addr)

	// Add the peer channel to the Peers map.
	msgCh := make(chan Message)
	Peers.Lock()
	Peers.m[addr] = msgCh
	Peers.Unlock()

	// Remove the peer when this function exits.
	defer func() {
		Peers.Lock()
		delete(Peers.m, addr)
		Peers.Unlock()
	}()

	// Send messages to the peer until an error occurs.
	enc := json.NewEncoder(c)
	for msg := range msgCh {
		err := enc.Encode(msg)
		if err != nil {
			log.Println(err)
			return
		}
	}
}

func accept(l net.Listener) {
	for {
		c, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go serve(c)
	}
}

func serve(c net.Conn) {
	defer c.Close()
	dec := json.NewDecoder(c)
	for {
		// Decode a message from the connection.
		var msg Message
		err := dec.Decode(&msg)
		if err != nil {
			if err != io.EOF {
				log.Println(err)
			}
			return
		}

		// Drop this message if we've seen a message with this ID
		// before and, if not, record the ID for future reference.
		Messages.Lock()
		seen := Messages.m[msg.ID]
		if !seen {
			Messages.m[msg.ID] = true
		}
		Messages.Unlock()
		if seen {
			continue
		}

		if msg.TTL > 0 {
			msg.TTL--
			broadcast(msg)
		}

		fmt.Println(msg.Body)
	}
}

func broadcast(m Message) {
	Peers.RLock()
	for _, ch := range Peers.m {
		go func(ch chan<- Message) {
			select {
			case ch <- m:
			case <-time.After(sendTimeout):
			}
		}(ch)
	}
	Peers.RUnlock()
}
