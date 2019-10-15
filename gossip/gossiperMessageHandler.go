package gossip

import (
	"github.com/somecookie/Peerster/helper"
	"github.com/somecookie/Peerster/packet"
	"net"
)

func (g *Gossiper) GossipPacketHandler(receivedPacket *packet.GossipPacket, peerAddr *net.UDPAddr) {

	if g.simple {
		if receivedPacket.Simple != nil {
			go g.SimpleMessageRoutine(receivedPacket.Simple, peerAddr)
		}
	} else {
		if receivedPacket.Rumor != nil {
			go g.RumorMessageRoutine(receivedPacket.Rumor, peerAddr)
		} else if receivedPacket.Status != nil {
			go g.StatusPacketRoutine(receivedPacket.Status, peerAddr)
		}
	}

}

//SimpleMessageRoutine handle the GossipPackets of type SimpleMessage
//It first prints the message and g's Peers.
//Finally it forwards message to all g's Peers (except peerAddr)
func (g *Gossiper) SimpleMessageRoutine(message *packet.SimpleMessage, peerAddr *net.UDPAddr) {
	packet.OutputSimpleMessage(message)
	g.ListPeers()
	message.RelayPeerAddr = g.gossipAddr
	g.Peers.Mutex.Lock()
	for _, addr := range g.Peers.List {
		if addr == nil || addr.String() == peerAddr.String() {
			continue
		}
		packetBytes, err := packet.GetPacketBytes(&packet.GossipPacket{Simple: message})

		if err == nil {
			_, err := g.connGossip.WriteToUDP(packetBytes, addr)
			helper.LogError(err)
		}
	}
	g.Peers.Mutex.Unlock()

}

//RumorMessageRoutine handles the RumorMessage.
//It first prints the message and g's Peers. Then it sends an ack to the peer that send the rumor.
//Finally, if it is a new Rumor g starts Rumormongering
func (g *Gossiper) RumorMessageRoutine(message *packet.RumorMessage, peerAddr *net.UDPAddr) {
	packet.OutputInRumorMessage(message, peerAddr)
	g.ListPeers()
	g.RumorState.Mutex.Lock()
	nextID := g.GetNextID(message.Origin)

	if message.ID >= nextID && message.Origin != g.Name {
		g.UpdateRumorState(message)
		g.sendStatusPacket(peerAddr)
		g.RumorState.Mutex.Unlock()
		g.Rumormongering(message, false, peerAddr, nil)
	} else {
		g.sendStatusPacket(peerAddr)
		g.RumorState.Mutex.Unlock()
	}
}

//StatusPacketRoutine handles the incoming StatusPacket.
//It first acknowledges the message  given the incoming vectorClock.
//Then it compares its own vector clock with the one in the StatusPacket.
//It either send a packet to the peer if it is missing one or ask for a packet with a StatusPacket.
//If both peer are in sync, g toss a coin and either stop the rumormongering or continue with a new peer.
func (g *Gossiper) StatusPacketRoutine(statusPacket *packet.StatusPacket, peerAddr *net.UDPAddr) {
	packet.OutputStatusPacket(statusPacket, peerAddr)
	g.ListPeers()
	b := g.AckRumors(peerAddr, statusPacket)

	if !b {
		g.StatusPacketHandler(statusPacket.Want, peerAddr, nil)
	}

}

//sendStatusPacket sends a StatusPacket to peerAddr that serves as an ACK to the RumorMessage.
func (g *Gossiper) sendStatusPacket(peerAddr *net.UDPAddr) {
	gossipPacket := &packet.GossipPacket{
		Status: &packet.StatusPacket{Want: g.RumorState.VectorClock},
	}
	g.sendMessage(gossipPacket, peerAddr)
}
