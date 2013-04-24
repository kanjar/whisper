// Package master provides functions for communicating with a whispernet master
// server.
package master

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
)

var masterAddr = flag.String("master", "", "master address")

// RegisterPeer registers a peer address with the master.
func RegisterPeer(addr string) error {
	if *masterAddr == "" {
		return flagErr
	}
	u := fmt.Sprintf("http://%s/hello", *masterAddr)
	v := url.Values{"addr": []string{addr}}
	r, err := http.PostForm(u, v)
	if err != nil {
		return fmt.Errorf("RegisterPeer: %v", err)
	}
	r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return fmt.Errorf("RegisterPeer: %v", r.Status)
	}
	return nil
}

// ListPeer retrieves a list of peer addresses from the master.
func ListPeers() ([]string, error) {
	if *masterAddr == "" {
		return nil, flagErr
	}
	u := fmt.Sprintf("http://%s/peers", *masterAddr)
	r, err := http.Get(u)
	if err != nil {
		return nil, fmt.Errorf("ListPeers: %v", err)
	}
	defer r.Body.Close()
	var peers []string
	err = json.NewDecoder(r.Body).Decode(&peers)
	if err != nil {
		return nil, fmt.Errorf("ListPeers: %v", err)
	}
	return peers, nil
}

var flagErr = errors.New(`You must specify the -master flag.
(And make sure you call flag.Parse() at the top of your main() function.)`)
