package gossip

import (
	"github.com/somecookie/Peerster/packet"
	"sync/atomic"
)

//HandleMessage is used to handle the messages that come from the client
func (g *Gossiper) HandleMessage(message *packet.Message) {
	packet.OutputMessage(message)
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
	g.RumorState.Mutex.Lock()
	g.UpdateRumorState(rumorMessage)
	g.RumorState.Mutex.Unlock()
	if len(g.Peers.List) > 0 {
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
	g.Peers.Mutex.Lock()
	for _, addr := range g.Peers.List {
		if addr == nil {
			continue
		}
		g.sendMessage(&packet.GossipPacket{Simple: simpleMessage}, addr)
	}
	g.Peers.Mutex.Unlock()
}
