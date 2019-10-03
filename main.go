package main

import (
	"Peerster/gossiper"
	"Peerster/helper"
	"flag"
	"net"
	"strings"
)

var g *gossiper.Gossiper

func init() {
	uiPort := flag.String( "UIPort", "8080", "port for the UI client (default \"8080\")")
	gossipAddr := flag.String("gossipAddr", "127.0.0.1:5000", "ip:port for the gossiper (default \"127.0.0.1:5000\"")
	name := flag.String("name", "default", "name of the gossiper")
	peersStr := flag.String("peers", "", "comma separated list of peers of the form ip:port")
	simple := flag.Bool("simple", false, "run gossiper in simple broadcast mode")

	handleFlags(peersStr, gossipAddr, uiPort, name, simple)

}

func handleFlags(peersStr *string, gossipAddr *string, uiPort *string, name *string, simple *bool) {
	peers := getPeersAddr(*peersStr)
	ipPort := strings.Split(*gossipAddr, ":")
	if len(ipPort) != 2 {
		helper.HandleCrashingErr(&helper.IllegalArgumentError{
			ErrorMessage: "gossipAddress has the wrong format",
			Where:        "main.go",
		})
	}

	var err error
	g, err = gossiper.BasicGossiperFactory(*uiPort, ipPort[1], ipPort[0], *name, peers, *simple)
	if err != nil {
		helper.HandleCrashingErr(err)
	}
}

func getPeersAddr(peersStr string) []*net.UDPAddr {
	tab := strings.Split(peersStr, ",")
	if len(tab) == 1 && tab[0] == "" {
		return make([]*net.UDPAddr, 10)
	}

	peers := make([]*net.UDPAddr, 10)
	for _, peer := range tab{
		udpAddr, err := net.ResolveUDPAddr("udp4", peer)
		if err == nil{
			peers = append(peers, udpAddr)
		}
	}

	return peers
}

func main() {
	g.HandleUDPClient()
}


