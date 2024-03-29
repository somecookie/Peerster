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
var rtimer int
var antiEntropy int
var hoplimit int
var N int
var stubbornTimeout int
var ackAll bool
var hw3ex2 bool
var hw3ex3 bool

func init() {
	uiPort := flag.String("UIPort", "8080", "port for the UI client (default \"8080\")")
	guiPort = *flag.String("GUIPort", "8080", "Port for the graphical interface (default \"8080\")")
	gossipAddr := flag.String("gossipAddr", "127.0.0.1:5000", "ip:port for the gossip (default \"127.0.0.1:5000\"")
	name := flag.String("name", "default", "name of the gossip")
	peersStr := flag.String("peers", "", "comma separated list of peers of the form ip:port")
	simple := flag.Bool("simple", false, "run gossip in simple broadcast mode")
	flag.IntVar(&antiEntropy,"antiEntropy", 10, "time in seconds for the anti-entropy (default 10 seconds)")
	flag.IntVar(&rtimer,"rtimer", 0, "Timeout in seconds to send route rumors. 0 (default) means disable sending route rumors")
	flag.IntVar(&hoplimit,"hoplimit", 10, "Hoplimit for the TLCMessage")
	flag.IntVar(&N,"N", 1, "Number of gossipers of the network")
	flag.IntVar(&stubbornTimeout,"stubbornTimeout", 5, "Time in seconds before a gossiper stubbornly resend a TLCMessage")
	flag.BoolVar(&runGUI, "runGUI", false, "allow to access a gui from this gossiper")
	flag.BoolVar(&ackAll, "ackAll", false, "run as in hmw3ex2")
	flag.BoolVar(&hw3ex2, "hw3ex2", false, "???")
	flag.BoolVar(&hw3ex3, "hw3ex3", false, "???")
	flag.Parse()
	handleFlags(*peersStr, *gossipAddr, *uiPort, *name, *simple)


}

func handleFlags(peersStr string, gossipAddr string, uiPort string, name string, simple bool) {
	peers := getPeersAddr(peersStr, gossipAddr)
	var err error
	g, err = gossip.GossiperFactory(gossipAddr, uiPort, name, peers, simple, ackAll, antiEntropy, rtimer, hoplimit, N,stubbornTimeout)
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

	go g.GossiperListener()

	if antiEntropy > 0{
		go g.AntiEntropyRoutine()
	}

	if rtimer > 0{
		go g.RouteRumorRoutine()
	}


	if runGUI {
		go HandleServerGUI()
	}
	g.ClientListener()

}
