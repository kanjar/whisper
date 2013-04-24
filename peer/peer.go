// This is a sample implementation of the final project for the "Whispering
// Gophers" code lab.
package main

import (
	"bufio"
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
	sendTimeout     = 1 * time.Second
	defaultTTL      = 5
)

type Message struct {
	ID   string
	Body string
	TTL  int
}

// Peers tracks the connected peers.
var Peers = struct {
	sync.RWMutex
	m map[string]chan<- Message
}{m: make(map[string]chan<- Message)}

// Messages tracks any messages we have seen.
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

	self := l.Addr().String()
	err = master.RegisterPeer(self)
	if err != nil {
		log.Fatal(err)
	}
	go poll(self)

	readInput()
}

// accept accepts connections from peers from the given listener.
func accept(l net.Listener) {
	for {
		c, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go readMessages(c)
	}
}

// poll periodically fetches a peer list from the master and connects to any
// new peers.
func poll(self string) {
	for {
		addrs, err := master.ListPeers()
		if err != nil {
			log.Println(err)
			continue
		}
		for _, addr := range addrs {
			// Don't connect to self.
			if addr == self {
				continue
			}

			// Don't connect if we're already connected.
			Peers.RLock()
			_, ok := Peers.m[addr]
			Peers.RUnlock()
			if ok {
				continue
			}

			go connect(addr)
		}
		time.Sleep(refreshInterval)
	}
}

// connect connects to the specified peer, add a message channel to the Peers
// map, and encodes any messages sent to that channel as JSON messages that it
// writes to the peer. When the peer connection goes down, the channel is
// removed from the Peers map.
func connect(peerAddr string) {
	// Set up TCP connection.
	c, err := net.Dial("tcp", peerAddr)
	if err != nil {
		log.Println(err)
		return
	}
	defer c.Close()

	// Some diagnostics.
	log.Println("- connected to", peerAddr)
	defer log.Println("- disconnected from", peerAddr)

	// Add the peer channel to the Peers map.
	msgCh := make(chan Message)
	Peers.Lock()
	Peers.m[peerAddr] = msgCh
	Peers.Unlock()

	// Remove the peer when this function exits.
	defer func() {
		Peers.Lock()
		delete(Peers.m, peerAddr)
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

// readMessages reads Messages from the given reader, re-broadcasts them
// to all connected peers, and logs them to the console.
func readMessages(rc io.ReadCloser) {
	defer rc.Close()
	dec := json.NewDecoder(rc)
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

		// Decrease message TTL and broadcast.
		if msg.TTL > 0 {
			msg.TTL--
			broadcast(msg)
		}

		fmt.Println(">", msg.Body)
	}
}

// readInput reads standard input and broadcasts each line as a message.
func readInput() {
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		id := helper.RandomID()
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

// broadcast sends a Message to all connected peers.
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
