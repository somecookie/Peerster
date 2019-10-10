package gossip

import (
	"Peerster/packet"
)

//handleMessage is used to handle the messages that come from the client
func (g *Gossiper) handleMessage(message *packet.Message) {
	packet.OutputMessage(message)
	if g.simple {
		g.sendSimpleMessage(message)
	} else {
		g.startRumor(message)
	}
}

//startRumor starts a new rumor when the gossiper receives a message from the client
func (g *Gossiper) startRumor(message *packet.Message) {
	if len(g.peers) > 0{
		g.counter++
		rumorMessage := &packet.RumorMessage{
			Origin: g.name,
			ID:     g.counter,
			Text:   message.Text,
		}
		packetToSend := &packet.GossipPacket{Rumor: rumorMessage}
		addr := g.selectPeerAtRandom()
		packet.OutputOutRumorMessage(addr)
		g.sendMessage(packetToSend, addr)
	}
}

//sendSimpleMessage handles the client's message when the simple flag is up
//It transform the Message in SimpleMessage and sends it to all its peers
func (g *Gossiper) sendSimpleMessage(message *packet.Message) {
	simpleMessage := &packet.SimpleMessage{
		OriginalName:  g.name,
		RelayPeerAddr: g.gossipAddr,
		Contents:      message.Text,
	}
	for _, addr := range g.peers {
		if addr == nil {
			continue
		}
		g.sendMessage(&packet.GossipPacket{Simple: simpleMessage}, addr)
	}
}
