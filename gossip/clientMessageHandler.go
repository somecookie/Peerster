package gossip

import (
	"github.com/somecookie/Peerster/packet"
	"sync/atomic"
)

//HandleMessage is used to handle the messages that come from the client
func (g *Gossiper) HandleMessage(message *packet.Message) {

	packet.PrintClientMessage(message)
	if g.simple {
		go g.sendSimpleMessage(message)
	} else {
		if message.Destination == nil{
			go g.startRumor(message)
		}else{
			if *message.Destination == g.Name{
				return
			}
			go g.startPrivate(message)
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
	g.sendMessage(&packet.GossipPacket{Private:pm}, g.DSDV.NextHop[*message.Destination])
	g.DSDV.Mutex.RUnlock()

	g.State.Mutex.Lock()
	g.State.UpdatePrivateQueue(*message.Destination, pm)
	g.State.Mutex.Unlock()
}
