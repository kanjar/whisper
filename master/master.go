package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

const pollInterval = 5 * time.Second

var Peers struct {
	sync.RWMutex
	s []string
}

func main() {
	go poller()
	http.HandleFunc("/hello", hello)
	http.HandleFunc("/peers", peers)
	http.ListenAndServe(":8000", nil)
}

func hello(w http.ResponseWriter, r *http.Request) {
	addr := r.FormValue("addr")
	Peers.Lock()
	Peers.s = append(Peers.s, addr)
	Peers.Unlock()
}

func peers(w http.ResponseWriter, r *http.Request) {
	Peers.RLock()
	b, err := json.Marshal(Peers.s)
	Peers.RUnlock()
	if err != nil {
		log.Println(err)
		return
	}
	w.Write(b)
}

func poller() {
	for {
		Peers.RLock()
		for _, addr := range Peers.s {
			go poll(addr)
		}
		Peers.RUnlock()
		time.Sleep(pollInterval)
	}
}

func poll(addr string) {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		Peers.Lock()
		for i, a := range Peers.s {
			if addr == a {
				copy(Peers.s[i:], Peers.s[i+1:])
				Peers.s = Peers.s[:len(Peers.s)-1]
				break
			}
		}
		Peers.Unlock()
		return
	}
	c.Close()
}
