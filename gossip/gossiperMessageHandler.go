package gossip

import (
	"github.com/somecookie/Peerster/helper"
	"github.com/somecookie/Peerster/packet"
	"math/rand"
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
		}else if receivedPacket.Private != nil{
			go g.PrivateMessageRoutine(receivedPacket.Private, peerAddr)
		}
	}

}

//SimpleMessageRoutine handle the GossipPackets of type SimpleMessage
//It first prints the message and g's Peers.
//Finally it forwards message to all g's Peers (except peerAddr)
func (g *Gossiper) SimpleMessageRoutine(message *packet.SimpleMessage, peerAddr *net.UDPAddr) {
	packet.PrintSimpleMessage(message)

	g.Peers.Mutex.RLock()
	defer g.Peers.Mutex.RUnlock()
	PrintPeers(g)

	message.RelayPeerAddr = g.GossipAddr
	for addrStr, addr := range g.Peers.Set {
		if addr == nil || addrStr == peerAddr.String() {
			continue
		}
		packetBytes, err := packet.GetPacketBytes(&packet.GossipPacket{Simple: message})

		if err == nil {
			_, err := g.connGossip.WriteToUDP(packetBytes, addr)
			helper.LogError(err)
		}
	}

}

//RumorMessageRoutine handles the RumorMessage.
//It first prints the message and g's Peers. Then it sends an ack to the peer that send the rumor.
//Finally, if it is a new Rumor g starts Rumormongering
func (g *Gossiper) RumorMessageRoutine(message *packet.RumorMessage, peerAddr *net.UDPAddr) {
	packet.PrintRumorMessage(message, peerAddr)

	g.Peers.Mutex.RLock()
	PrintPeers(g)
	g.Peers.Mutex.RUnlock()

	nextID := g.GetNextID(message.Origin)

	if message.ID >= nextID && message.Origin != g.Name {

		g.DSDV.Mutex.Lock()
		g.DSDV.Update(message, peerAddr)
		g.DSDV.Mutex.Unlock()

		g.UpdateRumorState(message)

		g.RumorState.Mutex.RLock()
		g.sendStatusPacket(peerAddr)
		g.RumorState.Mutex.RUnlock()

		g.Rumormongering(message, false, peerAddr, nil)
	} else {
		g.RumorState.Mutex.RLock()
		g.sendStatusPacket(peerAddr)
		g.RumorState.Mutex.RUnlock()
	}
}

//StatusPacketRoutine handles the incoming StatusPacket.
//It first acknowledges the message  given the incoming vectorClock.
//Then it compares its own vector clock with the one in the StatusPacket.
//It either send a packet to the peer if it is missing one or ask for a packet with a StatusPacket.
//If both peer are in sync, g toss a coin and either stop the rumormongering or continue with a new peer.
func (g *Gossiper) StatusPacketRoutine(statusPacket *packet.StatusPacket, peerAddr *net.UDPAddr) {
	packet.PrintStatusPacket(statusPacket, peerAddr)

	g.Peers.Mutex.RLock()
	PrintPeers(g)
	g.Peers.Mutex.RUnlock()

	g.pendingACK.Mutex.RLock()
	b := g.AckRumors(peerAddr, statusPacket)
	g.pendingACK.Mutex.RUnlock()

	if !b {
		g.StatusPacketHandler(statusPacket.Want, peerAddr, nil)
	}

}

func (g *Gossiper) StatusPacketHandler(peerVector []packet.PeerStatus, peerAddr *net.UDPAddr, rumorMessage *packet.RumorMessage) {
	//Check if S has messages that R has not seen yet
	g.RumorState.Mutex.RLock()
	b, msg := g.HasOther(g.RumorState.VectorClock, peerVector)
	g.RumorState.Mutex.RUnlock()

	if b {
		g.Rumormongering(msg, false, nil, peerAddr)
		return
	}
	//Check if R has messages that S has not seen yet
	g.RumorState.Mutex.RLock()
	b, _ = g.HasOther(peerVector, g.RumorState.VectorClock)
	g.RumorState.Mutex.RUnlock()

	if b {
		g.RumorState.Mutex.RLock()
		g.sendStatusPacket(peerAddr)
		g.RumorState.Mutex.RUnlock()
		return
	}

	packet.PrintInSync(peerAddr)

	if rand.Int()%2 == 0 && rumorMessage != nil {
		g.Rumormongering(rumorMessage, true, peerAddr, nil)
	}
}

//sendStatusPacket sends a StatusPacket to peerAddr that serves as an ACK to the RumorMessage.
func (g *Gossiper) sendStatusPacket(peerAddr *net.UDPAddr) {
	gossipPacket := &packet.GossipPacket{
		Status: &packet.StatusPacket{Want: g.RumorState.VectorClock},
	}
	g.sendMessage(gossipPacket, peerAddr)
}

//PrivateMessageRoutine handles the private messages.
func (g *Gossiper) PrivateMessageRoutine(privateMessage *packet.PrivateMessage, from *net.UDPAddr) {
	if privateMessage.Destination == g.Name{
		packet.PrintPrivateMessage(privateMessage)
	}else if privateMessage.HopLimit > 0{
		pm:= &packet.PrivateMessage{
			Origin:      g.Name,
			ID:          0,
			Text:        privateMessage.Text,
			Destination: privateMessage.Destination,
			HopLimit:    privateMessage.HopLimit - 1,
		}

		g.DSDV.Mutex.RLock()
		g.sendMessage(&packet.GossipPacket{Private:pm}, g.DSDV.NextHop[privateMessage.Destination])
		g.DSDV.Mutex.RUnlock()
	}
}
