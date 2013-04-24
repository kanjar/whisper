package main

import (
	"bufio"
	"encoding/json"
	"os"
)

type Message struct {
	Body string
}

func main() {
	r := bufio.NewReader(os.Stdin)
	e := json.NewEncoder(os.Stdout)
	for {
		s, err := r.ReadString('\n')
		if err != nil {
			break
		}
		m := Message{Body: s}
		err = e.Encode(m)
		if err != nil {
			break
		}
	}
}
