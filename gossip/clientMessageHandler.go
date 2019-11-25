package gossip

import (
	"github.com/somecookie/Peerster/packet"
	"strings"
	"sync/atomic"
)

//HandleMessage is used to handle the messages that come from the client
func (g *Gossiper) HandleMessage(message *packet.Message) {

	if message.Destination == nil && message.File == nil && message.Request == nil && message.Text != ""{
		packet.PrintClientMessage(message)
		if g.simple{
			go g.sendSimpleMessage(message)
		}else{
			go g.startRumor(message)
		}

	}else if message.Destination != nil && message.File == nil && message.Request == nil && message.Text != ""{
		packet.PrintClientMessage(message)
		go g.startPrivate(message)
	}else if message.Destination == nil && message.File != nil && message.Request == nil && message.Text == ""{
		packet.PrintClientMessage(message)
		go g.IndexFile(*message.File)
	}else if  message.Destination != nil && message.File != nil && message.Request != nil && message.Text == ""{
		packet.PrintClientMessage(message)
		go g.startDownload(message)
	}else if message.Destination == nil && message.File == nil && message.Request == nil && message.Text == "" && message.Keywords != nil{
		keywords := strings.Split(*message.Keywords, ",")
		if message.Budget != nil{
			sr := &packet.SearchRequest{
				Origin:   g.Name,
				Budget:   *message.Budget,
				Keywords: keywords,
			}
			go g.SearchRequestRoutine(sr, keywords)
		}else{
			//TODO: do thing page 3
		}
	}
}


//startRumor starts a new rumor when the gossiper receives a message from the client which is not a private message.
func (g *Gossiper) startRumor(message *packet.Message) {

	atomic.AddUint32(&g.counter, 1)
	rumorMessage := &packet.RumorMessage{
		Origin: g.Name,
		ID:     g.counter,
		Text:   message.Text,
	}
	g.State.UpdateGossiperState(rumorMessage)

	g.Peers.Mutex.RLock()
	length := len(g.Peers.Set)
	g.Peers.Mutex.RUnlock()


	if length > 0 {
		g.Rumormongering(rumorMessage, false, nil, nil)
	}
}

//sendSimpleMessage handles the client's message when the simple flag is up
//It transform the Message in SimpleMessage and sends it to all its Peers
func (g *Gossiper) sendSimpleMessage(message *packet.Message) {
	simpleMessage := &packet.SimpleMessage{
		OriginalName:  g.Name,
		RelayPeerAddr: g.GossipAddr,
		Contents:      message.Text,
	}

	g.Peers.Mutex.RLock()
	defer g.Peers.Mutex.RUnlock()

	for _,addr := range g.Peers.Set {
		if addr == nil {
			continue
		}
		g.sendMessage(&packet.GossipPacket{Simple: simpleMessage}, addr)
	}
}

//startPrivate starts a private chat between g.Name and message.Destination
func (g *Gossiper) startPrivate(message *packet.Message) {

	pm:= &packet.PrivateMessage{
		Origin:      g.Name,
		ID:          0,
		Text:        message.Text,
		Destination: *message.Destination,
		HopLimit:    packet.MaxHops - 1,
	}

	g.DSDV.Mutex.RLock()
	if g.DSDV.Contains(*message.Destination){
		g.sendMessage(&packet.GossipPacket{Private:pm}, g.DSDV.NextHop[*message.Destination])
		g.DSDV.Mutex.RUnlock()

		g.State.Mutex.Lock()
		g.State.UpdatePrivateQueue(*message.Destination, pm)
		g.State.Mutex.Unlock()
	}else{
		g.DSDV.Mutex.RUnlock()
	}

}

