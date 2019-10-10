package gossip

import (
	"Peerster/helper"
	"Peerster/packet"
	"net"
)

func (g *Gossiper) handleGossipPacket(receivedPacket *packet.GossipPacket, peerAddr *net.UDPAddr){
	if receivedPacket.Rumor != nil{
		g.handleRumorMessage(receivedPacket.Rumor, peerAddr)
	}else if receivedPacket.Simple != nil && g.simple{
		g.handleSimpleMessage(receivedPacket.Simple, peerAddr)
	}else if receivedPacket.Status != nil{

	}

}

//handleSimpleMessage handle the GossipPackets of type SimpleMessage
func (g *Gossiper) handleSimpleMessage(message *packet.SimpleMessage, peerAddr *net.UDPAddr) {
	packet.OutputSimpleMessage(message)
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

func (g *Gossiper) handleRumorMessage(message *packet.RumorMessage, peerAddr *net.UDPAddr) {
	if message.ID >= g.GetNextID(message.Origin) && message.Origin != g.name{
		packet.OutputInRumorMessage(message, peerAddr)
		g.UpdateVectorClock(message.Origin, message.ID)
		g.UpdateArchive(message)
		peerAddr := g.selectPeerAtRandom()
		g.sendMessage(&packet.GossipPacket{Rumor:message}, peerAddr)
		g.ListPeers()
	}
}