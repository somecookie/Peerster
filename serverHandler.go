package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
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
		name, _ := request.URL.Query()["name"]
		var jsonValue []byte
		var err error

		if name[0] == "Rumors"{
			g.State.Mutex.RLock()

			messages := make([]struct{
				Origin string
				Text string
			},0)
			for _, msg := range g.State.RumorQueue{
				if msg.TLCMessage != nil{
					messages = append(messages, struct {
						Origin string
						Text   string
					}{Origin: msg.TLCMessage.Origin, Text: fmt.Sprintf("%s confirmed", msg.TLCMessage.TxBlock.Transaction.Name)})
				}else{
					messages = append(messages, struct {
						Origin string
						Text   string
					}{Origin: msg.Rumor.Origin, Text: msg.Rumor.Text})
				}

			}
			jsonValue, err = json.Marshal(messages)
			g.State.Mutex.RUnlock()
		}else{
			g.State.Mutex.RLock()
			messages, ok := g.State.PrivateQueue[name[0]]
			g.State.Mutex.RUnlock()

			if ok{
				jsonValue, err = json.Marshal(messages)
			}else{
				empty := make([]packet.PrivateMessage,0)
				jsonValue, err = json.Marshal(empty)
			}

		}

		if err == nil {
			w.WriteHeader(http.StatusOK)
			w.Write(jsonValue)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	case "POST":
		err := request.ParseForm()
		if err == nil {
			text := request.Form.Get("value")
			dest := request.Form.Get("dest")
			rm := &packet.Message{Text:text, Destination:&dest }


			if dest == "Rumors"{
				rm.Destination = nil
			}



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

func downloadHandler(writer http.ResponseWriter, request *http.Request) {
	err := request.ParseForm()
	if err == nil {
		metahashStr := request.Form.Get("metahash")
		dest := request.Form.Get("from")
		fileName := request.Form.Get("fileName")
		
		request, err := hex.DecodeString(metahashStr)
		
		if err != nil || len(request) != 32{
			writer.WriteHeader(http.StatusBadRequest)
		}else{
			writer.WriteHeader(http.StatusOK)
			g.HandleMessage(&packet.Message{
				Text:        "",
				Destination: &dest,
				File:        &fileName,
				Request:     &request,
			})
		}

	}
}

func searchHandler(writer http.ResponseWriter, request *http.Request) {
	enableCors(&writer)
	switch request.Method {
	case "POST":
		if err := request.ParseForm(); err == nil{
			keywords := request.Form.Get("keywords")
			fmt.Println(keywords)
			message := &packet.Message{Keywords:&keywords}
			g.StartSearchRequest(message)

		}else{
			writer.WriteHeader(http.StatusInternalServerError)
		}

	default:
		writer.WriteHeader(http.StatusNotFound)
	}
}
func matchesHandler(w http.ResponseWriter, request *http.Request) {
	enableCors(&w)
	switch request.Method{
	case "GET":
		g.Matches.RLock()
		matchesAsJSON, err := json.Marshal(g.Matches.Queue)
		g.Matches.RUnlock()
		if err == nil {
			w.WriteHeader(http.StatusOK)
			w.Write(matchesAsJSON)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	default:
		w.WriteHeader(http.StatusNotFound)

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
	http.HandleFunc("/download", downloadHandler)
	http.HandleFunc("/search", searchHandler)
	http.HandleFunc("/matches", matchesHandler)
	for {
		err := http.ListenAndServe(serverAddr, nil)
		helper.LogError(err)
	}
}






