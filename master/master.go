package main

import (
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

const pollInterval = 5 * time.Second

var (
	httpAddr    = flag.String("http", ":8000", "HTTP listen address")
	pollEnabled = flag.Bool("poll", true, "Poll for dead peers")
)

var Peers struct {
	sync.RWMutex
	s []string
}

func main() {
	flag.Parse()
	if *pollEnabled {
		go poller()
	}
	http.HandleFunc("/", root)
	http.HandleFunc("/hello", hello)
	http.HandleFunc("/peers", peers)
	http.ListenAndServe(*httpAddr, nil)
}

func root(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	Peers.RLock()
	peers := append([]string{}, Peers.s...)
	Peers.RUnlock()
	err := rootTemplate.Execute(w, peers)
	if err != nil {
		log.Println(err)
	}
}

var rootTemplate = template.Must(template.New("root").Parse(`
<!DOCTYPE html>
<html>
	<head>
		<meta http-equiv="refresh" content="1">
	</head>
	<body>
		<h3>Connected peers:</h3>
		<ul>
		{{range .}}
		<li>{{.}}</li>
		{{end}}
		</ul>
	</body>
</html>
`))

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
