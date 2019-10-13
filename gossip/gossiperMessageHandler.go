package gossip

import (
	"github.com/somecookie/Peerster/helper"
	"github.com/somecookie/Peerster/packet"
	"math/rand"
	"net"
)

func (g *Gossiper) handleGossipPacket(receivedPacket *packet.GossipPacket, peerAddr *net.UDPAddr) {

	if receivedPacket.Rumor != nil {
		go g.handleRumorMessage(receivedPacket.Rumor, peerAddr)
	} else if receivedPacket.Simple != nil && g.simple {
		go g.handleSimpleMessage(receivedPacket.Simple, peerAddr)
	} else if receivedPacket.Status != nil {
		go g.handleStatusPacket(receivedPacket.Status, peerAddr)
	}

}

//handleSimpleMessage handle the GossipPackets of type SimpleMessage
//It first prints the message and g's peers.
//Finally it forwards message to all g's peers (except peerAddr)
func (g *Gossiper) handleSimpleMessage(message *packet.SimpleMessage, peerAddr *net.UDPAddr) {
	packet.OutputSimpleMessage(message)
	g.ListPeers()
	message.RelayPeerAddr = g.gossipAddr

	for _, addr := range g.peers {
		if addr == nil || addr.String() == peerAddr.String() {
			continue
		}
		packetBytes, err := packet.GetPacketBytes(&packet.GossipPacket{Simple: message})

		if err == nil {
			_, err := g.connGossip.WriteToUDP(packetBytes, addr)
			helper.LogError(err)
		}
	}

}

//handleRumorMessage handles the RumorMessage.
//It first prints the message and g's peers. Then it sends an ack to the peer that send the rumor.
//Finally, if it is a new Rumor g starts Rumormongering
func (g *Gossiper) handleRumorMessage(message *packet.RumorMessage, peerAddr *net.UDPAddr) {
	packet.OutputInRumorMessage(message, peerAddr)
	g.ListPeers()
	g.rumorState.Mutex.Lock()
	nextID := g.GetNextID(message.Origin)

	if message.ID >= nextID && message.Origin != g.name {
		g.UpdateRumorState(message)
		g.sendStatusPacket(peerAddr)
		g.rumorState.Mutex.Unlock()
		g.Rumormongering(message, false)
	}else{
		//g.sendStatusPacket(peerAddr)
		g.rumorState.Mutex.Unlock()
	}
}

//handleStatusPacket handles the incoming StatusPacket
//It first acknowledges the message  given the incoming vectorClock.
//Then it compares its own vector clock with the one in the StatusPacket.
//It either send a packet to the peer if it is missing one or ask for a packet with a StatusPacket.
//If both peer are in sync, g toss a coin and either stop the rumormongering or continue with a new peer.
func (g *Gossiper) handleStatusPacket(statusPacket *packet.StatusPacket, peerAddr *net.UDPAddr) {
	packet.OutputStatusPacket(statusPacket, peerAddr)
	g.ListPeers()

	peerVector := statusPacket.Want
	wantMessage := false
	rumorMessage := g.AckRumors(peerVector, peerAddr)

	//check if the sender has messages that the receiver has not seen
	g.rumorState.Mutex.Lock()
	for _, senderStatus := range g.rumorState.VectorClock {
		for _, receiverStatus := range peerVector {
			sameOrigin := senderStatus.Identifier == receiverStatus.Identifier
			if sameOrigin && senderStatus.NextID > receiverStatus.NextID {
				nextID := receiverStatus.NextID
				origin := receiverStatus.Identifier
				message := g.rumorState.ArchivedMessages[origin][nextID]
				g.sendMessage(&packet.GossipPacket{Rumor: message}, peerAddr)
				g.rumorState.Mutex.Unlock()
				return
			} else if sameOrigin && senderStatus.NextID < receiverStatus.NextID {
				wantMessage = true
			}
		}
	}

	//ask for missing packets to the receiver if possible
	if wantMessage {
		g.sendStatusPacket(peerAddr)
		g.rumorState.Mutex.Unlock()
		return
	}

	g.rumorState.Mutex.Unlock()

	packet.OutputInSync(peerAddr)
	if rand.Int()%2 == 0 {
		g.Rumormongering(rumorMessage, true)
	}

}

//sendStatusPacket sends a StatusPacket to peerAddr that serves as an ACK to the RumorMessage.
func (g *Gossiper) sendStatusPacket(peerAddr *net.UDPAddr) {
	gossipPacket := &packet.GossipPacket{
		Status: &packet.StatusPacket{Want: g.rumorState.VectorClock},
	}
	g.sendMessage(gossipPacket, peerAddr)
}
