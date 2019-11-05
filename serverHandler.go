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
		jsonValue, err := json.Marshal(g.Peers.PeersAsStringList())
		g.Peers.Mutex.RUnlock()

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

			if peerAddrStr == g.GossipAddr {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			udpAddr, err := net.ResolveUDPAddr("udp4", peerAddrStr)

			if err == nil {
				g.Peers.Mutex.Lock()
				g.Peers.Add(udpAddr)
				g.Peers.Mutex.Unlock()
			}

		}
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}



func rumorMessagesHandler(w http.ResponseWriter, request *http.Request) {
	enableCors(&w)
	switch request.Method {
	case "GET":
		g.State.Mutex.RLock()
		jsonValue, err := json.Marshal(g.State.RumorQueue)
		g.State.Mutex.RUnlock()

		if err == nil {
			w.WriteHeader(http.StatusOK)
			w.Write(jsonValue)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	case "POST":
		err := request.ParseForm()
		if err == nil {
			rm := &packet.Message{Text: request.Form.Get("value")}
			g.HandleMessage(rm)
		}
	default:
		w.WriteHeader(http.StatusNotFound)
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
		w.WriteHeader(http.StatusNotFound)

	}
}

func originHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	switch r.Method {
	case "GET":
		g.DSDV.Mutex.RLock()
		originsAsJSON, err := json.Marshal(g.DSDV.GetOrigins())
		g.DSDV.Mutex.RUnlock()
		if err == nil {
			w.WriteHeader(http.StatusOK)
			w.Write(originsAsJSON)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	default:
		w.WriteHeader(http.StatusNotFound)

	}
}

func shareFileHandler(writer http.ResponseWriter, request *http.Request) {
	enableCors(&writer)
	switch request.Method {
	case "POST":
		err := request.ParseForm()
		if err == nil {
			fileName := request.Form.Get("fileName")
			g.IndexFile(fileName)
		}

	default:
		writer.WriteHeader(http.StatusNotFound)
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
	http.HandleFunc("/origin", originHandler)
	http.HandleFunc("/shareFile", shareFileHandler)
	for {
		err := http.ListenAndServe(serverAddr, nil)
		helper.LogError(err)
	}
}

