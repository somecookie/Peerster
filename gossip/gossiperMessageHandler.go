package gossip

import (
	"Peerster/helper"
	"Peerster/packet"
	"fmt"
	"net"
)

func (g *Gossiper) handleGossipPacket(receivedPacket *packet.GossipPacket, peerAddr *net.UDPAddr){
	if receivedPacket.Rumor != nil{
		g.handleRumorMessage(receivedPacket.Rumor)
	}else if receivedPacket.Simple != nil && g.simple{
		g.handleSimpleMessage(receivedPacket.Simple, peerAddr)
	}else if receivedPacket.Status != nil{

	}

}

//handleSimpleMessage handle the GossipPackets of type SimpleMessage
func (g *Gossiper) handleSimpleMessage(message *packet.SimpleMessage, peerAddr *net.UDPAddr) {
	fmt.Printf("SIMPLE MESSAGE origin %s from %s contents %s\n", message.OriginalName, message.RelayPeerAddr, message.Contents)
	message.RelayPeerAddr = g.gossipAddr

	for _,addr := range g.peers {
		if addr == nil || addr.String() == peerAddr.String() {
			continue
		}
		packetBytes, err := packet.GetPacketBytes(&packet.GossipPacket{Simple: message})

		if err == nil {
			_, err := g.connGossip.WriteToUDP(packetBytes, addr)
			helper.LogError(err)
		}
	}
	g.ListPeers()
}

func (g *Gossiper) handleRumorMessage(message *packet.RumorMessage) {
	if message.ID > g.vectorClock[message.Origin]{
		g.vectorClock[message.Origin]++
		peerAddr := g.selectPeerAtRandom()
		g.sendMessage(&packet.GossipPacket{Rumor:message}, peerAddr)
		g.ListPeers()
	}
}