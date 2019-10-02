package main

import (
	"flag"
	"fmt"
	"github.com/rferreir/Peerster/gossiper"
	"strings"
)

var (
	uiPort     string
	gossipAddr string
	name       string
	peers      []string
	simple     bool
)

func init() {
	flag.StringVar(&uiPort, "UIPort", "8080", "port for the UI client (default \"8080\")")
	flag.StringVar(&gossipAddr, "gossipAddr", "127.0.0.1:5000", "ip:port for the g (default \"127.0.0.1:5000\"")
	flag.StringVar(&name, "name", "default", "name of the gossiper")
	peersStr := flag.String("peers", "", "comma separated list of peers of the form ip:port")
	flag.BoolVar(&simple, "simple", false, "run g in simple broadcast mode")
	flag.Parse()

	peers = strings.Split(*peersStr, ",")

	if len(peers) == 1 && peers[0] == ""{
		peers = make([]string,0)
	}




}

func main() {
	fmt.Println(&gossiper.Gossiper{
		UIPort:     uiPort,
		GossipAddr: gossipAddr,
		Name:       name,
		Peers:      peers,
		Simple:     simple,
	})
}
