package main

import (
	"Peerster/peer"
	"Peerster/utils"
	"flag"
	"fmt"
	"strings"
)

var (
	uiport     string
	gossipAddr string
	name       string
	peersStr   string
	peers      []string
	simple     bool
	gossiper   peer.Gossiper
)

func init() {
	flag.StringVar(&uiport, "UIPort", "8080", "port for the UI client (default \"8080\")")
	flag.StringVar(&gossipAddr, "gossipAddr", "127.0.0.1:5000", "ip:port for the gossiper (default \"127.0.0.1:5000\"")
	flag.StringVar(&name, "name", "default", "name of the gossiper")
	flag.StringVar(&peersStr, "peers", "", "comma separated list of peers of the form ip:port")
	flag.BoolVar(&simple, "simple", false, "run gossiper in simple broadcast mode")
	flag.Parse()

	if !utils.ValidPort(uiport) {
		panic("In Gossiper: UIPort is not a valid port")
	}

	if !utils.ValidPortIP(gossipAddr) {
		panic("In Gossiper: gossipAddr is not a valid pair ip:port")
	}

	peers = make([]string, 0)
	if len(peersStr) != 0 {
		splitPairs := strings.Split(peersStr, ",")

		for _, pair := range splitPairs {
			if utils.ValidPortIP(pair) {
				peers = append(peers, pair)
			}
		}
	}

	gossiper = peer.Gossiper{
		UIPort:     uiport,
		GossipAddr: gossipAddr,
		Name:       name,
		Peers:      peers,
		Simple:     simple,
	}

}

func main() {
	fmt.Println(&gossiper)
}
