package main

import (
	"encoding/json"
	"github.com/somecookie/Peerster/helper"
	"github.com/somecookie/Peerster/packet"
	"net"
	"net/http"
)

func nodeHandler(w http.ResponseWriter, request *http.Request) {
	enableCors(&w)
	enableCors(&w)
	switch request.Method {
	case "GET":
		g.Peers.Mutex.RLock()
		keys := make([]string, 0, len(g.Peers.Set))
		for k := range g.Peers.Set {
			keys = append(keys, k)
		}
		g.Peers.Mutex.RUnlock()

		jsonValue, err := json.Marshal(keys)
		if err == nil {
			w.WriteHeader(http.StatusOK)
			w.Write(jsonValue)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	case "POST":
		err := request.ParseForm()
		if err == nil {
			peerAddrStr := request.Form.Get("value")
			udpAddr, err := net.ResolveUDPAddr("udp4", peerAddrStr)

			if err == nil {
				g.Peers.Mutex.Lock()
				g.Peers.Add(udpAddr)
				g.Peers.Mutex.Unlock()
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
		g.RumorState.Mutex.RLock()
		rmlist := make([]packet.RumorMessage, 0, len(g.RumorState.MessageList))
		for _, msgptr := range g.RumorState.MessageList {
			rmlist = append(rmlist, *msgptr)
		}
		g.RumorState.Mutex.RUnlock()

		jsonValue, err := json.Marshal(rmlist)
		if err == nil {
			w.WriteHeader(http.StatusOK)
			w.Write(jsonValue)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	case "POST":
		err := request.ParseForm()
		if err == nil {
			rm := packet.Message{Text: request.Form.Get("value")}
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
		if err == nil {
			w.WriteHeader(http.StatusOK)
			w.Write(jsonValue)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}

	default:
		w.WriteHeader(404)

	}
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}

func HandleServerGUI() {
	serverAddr := "localhost:" + guiPort
	http.Handle("/", http.FileServer(http.Dir("./frontend")))
	http.HandleFunc("/id", IDHandler)
	http.HandleFunc("/message", rumorMessagesHandler)
	http.HandleFunc("/node", nodeHandler)
	for {
		err := http.ListenAndServe(serverAddr, nil)
		helper.LogError(err)
	}
}
