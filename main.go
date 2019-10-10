package main

import (
	"Peerster/gossip"
	"Peerster/helper"
	"flag"
	"net"
	"strings"
	"sync"
)

var g *gossip.Gossiper

func init() {
	uiPort := flag.String("UIPort", "8080", "port for the UI client (default \"8080\")")
	gossipAddr := flag.String("gossipAddr", "127.0.0.1:5000", "ip:port for the gossip (default \"127.0.0.1:5000\"")
	name := flag.String("name", "default", "name of the gossip")
	peersStr := flag.String("peers", "", "comma separated list of peers of the form ip:port")
	simple := flag.Bool("simple", false, "run gossip in simple broadcast mode")
	flag.Parse()
	handleFlags(*peersStr, *gossipAddr, *uiPort, *name, *simple)

}

func handleFlags(peersStr string, gossipAddr string, uiPort string, name string, simple bool) {
	peers := getPeersAddr(peersStr, gossipAddr)
	var err error
	g, err = gossip.BasicGossiperFactory(gossipAddr, uiPort, name, peers, simple)
	helper.HandleCrashingErr(err)
}

func getPeersAddr(peersStr, gossipAddr string) []*net.UDPAddr {
	tab := strings.Split(peersStr, ",")

	peers := make([]*net.UDPAddr,0)
	if len(tab) == 1 && tab[0] == "" {
		return peers
	}

	for _, addr := range tab {
		if addr == gossipAddr {
			continue
		}
		udpAddr, err := net.ResolveUDPAddr("udp4", addr)
		if err == nil {
			peers = append(peers, udpAddr)
		}
	}

	return peers
}

func main() {
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	go g.HandleUDPClient(&waitGroup)
	go g.HandleUPDGossiper(&waitGroup)
	waitGroup.Wait()
}
