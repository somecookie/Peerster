package main

import (
	"flag"
	"github.com/somecookie/Peerster/gossip"
	"github.com/somecookie/Peerster/helper"
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
	antiEntropy := flag.Int("antiEntropy", 10, "time in seconds for the anti-entropy (default 10 seconds)")
	flag.Parse()
	handleFlags(*peersStr, *gossipAddr, *uiPort, *name, *simple, *antiEntropy)

}

func handleFlags(peersStr string, gossipAddr string, uiPort string, name string, simple bool, antiEntropy int) {
	peers := getPeersAddr(peersStr, gossipAddr)
	if antiEntropy <= 0{
		antiEntropy = 10
	}
	var err error
	g, err = gossip.BasicGossiperFactory(gossipAddr, uiPort, name, peers, simple, antiEntropy)
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
	//rand.Seed(time.Now().UTC().Unix())
	waitGroup := sync.WaitGroup{}
	waitGroup.Add(2)
	go func(){
		defer waitGroup.Done()
		g.HandleUDPClient()
	}()

	go func(){
		defer waitGroup.Done()
		g.HandleUPDGossiper()
	}()
	waitGroup.Wait()
}
