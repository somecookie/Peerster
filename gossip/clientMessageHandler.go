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
		go g.startRumor(message)
	}
}

//startRumor starts a new rumor when the gossiper receives a message from the client
func (g *Gossiper) startRumor(message *packet.Message) {

	atomic.AddUint32(&g.counter, 1)
	rumorMessage := &packet.RumorMessage{
		Origin: g.Name,
		ID:     g.counter,
		Text:   message.Text,
	}
	g.UpdateRumorState(rumorMessage)

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
		RelayPeerAddr: g.gossipAddr,
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
