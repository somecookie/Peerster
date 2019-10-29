package main

import (
	"flag"
	"github.com/somecookie/Peerster/gossip"
	"github.com/somecookie/Peerster/helper"
	"net"
	"strings"
)

var g *gossip.Gossiper
var guiPort string
var runGUI bool

func init() {
	uiPort := flag.String("UIPort", "8080", "port for the UI client (default \"8080\")")
	guiPort = *flag.String("GUIPort", "8080", "Port for the graphical interface (default \"8080\")")
	gossipAddr := flag.String("gossipAddr", "127.0.0.1:5000", "ip:port for the gossip (default \"127.0.0.1:5000\"")
	name := flag.String("name", "default", "name of the gossip")
	peersStr := flag.String("peers", "", "comma separated list of peers of the form ip:port")
	simple := flag.Bool("simple", false, "run gossip in simple broadcast mode")
	antiEntropy := flag.Int("antiEntropy", 10, "time in seconds for the anti-entropy (default 10 seconds)")
	flag.BoolVar(&runGUI, "runGUI", false, "allow to access a gui from this gossiper")
	flag.Parse()
	handleFlags(*peersStr, *gossipAddr, *uiPort, *name, *simple, *antiEntropy)

}

func handleFlags(peersStr string, gossipAddr string, uiPort string, name string, simple bool, antiEntropy int) {
	peers := getPeersAddr(peersStr, gossipAddr)
	if antiEntropy <= 0 {
		antiEntropy = 10
	}
	var err error
	g, err = gossip.BasicGossiperFactory(gossipAddr, uiPort, name, peers, simple, antiEntropy)
	helper.HandleCrashingErr(err)
}

func getPeersAddr(peersStr, gossipAddr string) []*net.UDPAddr {
	tab := strings.Split(peersStr, ",")

	peers := make([]*net.UDPAddr, 0)
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

	go g.HandleUDPClient()
	if runGUI {
		go HandleServerGUI()
	}
	g.HandleUPDGossiper()
}
