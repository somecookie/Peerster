package gossip

import (
	"Peerster/packet"
	"fmt"
)

//handleMessage is used to handle the messages that come from the client
func (g *Gossiper) handleMessage(message *packet.Message) {
	fmt.Printf("CLIENT MESSAGE %s\n", message.Text)
	if g.simple {
		g.sendSimpleMessage(message)
	} else {
		g.startRumor(message)
	}
}

//startRumor starts a new rumor when the gossiper receives a message from the client
func (g *Gossiper) startRumor(message *packet.Message) {
	g.vectorClock[g.name]++
	rumorMessage := &packet.RumorMessage{
		Origin: g.name,
		ID:     g.vectorClock[g.name],
		Text:   message.Text,
	}
	packetToSend := &packet.GossipPacket{Rumor: rumorMessage}
	addr := g.selectPeerAtRandom()
	fmt.Printf("MONGERING with %s\n", addr.String())
	g.sendMessage(packetToSend, addr)
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
