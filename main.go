package main

import (
	"encoding/json"
	"flag"
	"github.com/somecookie/Peerster/gossip"
	"github.com/somecookie/Peerster/helper"
	"github.com/somecookie/Peerster/packet"
	"net"
	"net/http"
	"strings"
	"sync"
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
	flag.BoolVar(&runGUI,"runGUI", false, "allow to access a gui from this gossiper")
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
	//rand.Seed(time.Now().UTC().Unix())
	waitGroup := sync.WaitGroup{}
	waitGroup.Add(3)
	go func() {
		defer waitGroup.Done()
		g.HandleUDPClient()
	}()

	go func() {
		defer waitGroup.Done()
		g.HandleUPDGossiper()
	}()

	if runGUI{
		go func() {
			defer waitGroup.Done()
			serverAddr := "localhost:"+guiPort
			http.Handle("/", http.FileServer(http.Dir("./frontend")))
			http.HandleFunc("/id", IDHandler)
			http.HandleFunc("/message", rumorMessagesHandler)
			http.HandleFunc("/node", nodeHandler)

			for {
				err := http.ListenAndServe(serverAddr, nil)
				helper.LogError(err)
			}
		}()
	}

	waitGroup.Wait()

}

func nodeHandler(w http.ResponseWriter, request *http.Request) {
	enableCors(&w)
	enableCors(&w)
	switch request.Method {
	case "GET":
		addr := make([]string, 0)
		g.Peers.Mutex.Lock()
		for _, peerAddr := range g.Peers.List {
			addr = append(addr, peerAddr.String())
		}
		g.Peers.Mutex.Unlock()

		jsonValue, err := json.Marshal(addr)
		if err == nil{
			w.WriteHeader(http.StatusOK)
			w.Write(jsonValue)
		}else{
			w.WriteHeader(http.StatusInternalServerError)
		}
	case "POST":
		err := request.ParseForm()
		if err == nil{
			peerAddrStr := request.Form.Get("value")
			udpAddr, err := net.ResolveUDPAddr("udp4", peerAddrStr)

			if err == nil{
				g.AddPeer(udpAddr)
			}

		}
	default:
		w.WriteHeader(404)
	}
}

func rumorMessagesHandler(w http.ResponseWriter, request *http.Request) {
	enableCors(&w)
	switch request.Method {
	case "GET":
		rmlist := make([]packet.RumorMessage, 0)
		g.RumorState.Mutex.Lock()
		for _, msgptr := range g.RumorState.MessageList {
			rmlist = append(rmlist, *msgptr)
		}
		g.RumorState.Mutex.Unlock()

		jsonValue, err := json.Marshal(rmlist)
		if err == nil{
			w.WriteHeader(http.StatusOK)
			w.Write(jsonValue)
		}else{
			w.WriteHeader(http.StatusInternalServerError)
		}
	case "POST":
		err := request.ParseForm()
		if err == nil{
			rm := packet.Message{Text:request.Form.Get("value")}
			g.HandleMessage(&rm)
		}
	default:
		w.WriteHeader(404)
	}
}

func IDHandler(w http.ResponseWriter, request *http.Request) {
	enableCors(&w)
	switch request.Method {
	case "GET":
		jsonValue, err := json.Marshal(g.Name)
		if err == nil{
			w.WriteHeader(http.StatusOK)
			w.Write(jsonValue)
		}else{
			w.WriteHeader(http.StatusInternalServerError)
		}

	default:
		w.WriteHeader(404)

	}
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}
