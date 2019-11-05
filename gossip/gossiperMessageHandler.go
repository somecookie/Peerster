package gossip

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/somecookie/Peerster/helper"
	"github.com/somecookie/Peerster/packet"
	"math/rand"
	"net"
)

var hasher = sha256.New()

func (g *Gossiper) GossipPacketHandler(receivedPacket *packet.GossipPacket, from *net.UDPAddr) {
	if g.simple {
		if receivedPacket.Simple != nil {
			go g.SimpleMessageRoutine(receivedPacket.Simple, from)
		}
	} else {
		if receivedPacket.Rumor != nil {
			go g.RumorMessageRoutine(receivedPacket.Rumor, from)
		} else if receivedPacket.Status != nil {
			go g.StatusPacketRoutine(receivedPacket.Status, from)
		} else if receivedPacket.Private != nil {
			go g.PrivateMessageRoutine(receivedPacket.Private)
		} else if receivedPacket.DataRequest != nil {
			go g.DataRequestRoutine(receivedPacket.DataRequest)
		} else if receivedPacket.DataReply != nil {
			go g.DataReplyRoutine(receivedPacket.DataReply)
		}
	}

}

//DataReplyRoutine handles the incoming dataReply.
func (g *Gossiper) DataReplyRoutine(dataReply *packet.DataReply) {
	if dataReply.Destination == g.Name{

		hasher.Reset()
		_, err := hasher.Write(dataReply.Data)

		if err != nil {
			helper.LogError(err)
			return
		}
		hash := hasher.Sum(nil)
		receivedHashString := hex.EncodeToString(dataReply.HashValue)
		g.Requested.Mutex.RLock()
		if hex.EncodeToString(hash) == receivedHashString  || len(dataReply.Data) == 0 || dataReply.Data == nil{
			if origins, ok := g.Requested.ACKs[dataReply.Origin]; ok {
				if c, ok := origins[receivedHashString]; ok {
					c <- dataReply
				}
			}
		}
		g.Requested.Mutex.RUnlock()
	}else if dataReply.HopLimit > 0 {
		dataReply.HopLimit -= 1

		g.DSDV.Mutex.RLock()
		if g.DSDV.Contains(dataReply.Destination) {
			g.sendMessage(&packet.GossipPacket{DataReply: dataReply}, g.DSDV.NextHop[dataReply.Destination])
		}
		g.DSDV.Mutex.RUnlock()
	}

}

//DataRequestRoutine handles the incoming request.
//It either discards the packet when the hop-limit is 0,
//or if the destination is the gossiper, process the packet.
func (g *Gossiper) DataRequestRoutine(dataRequest *packet.DataRequest) {
	//fmt.Println(hex.EncodeToString(dataRequest.HashValue))
	if dataRequest.Destination == g.Name {
		g.FilesIndex.Mutex.RLock()
		chunk := g.FilesIndex.FindChunkFromHash(hex.EncodeToString(dataRequest.HashValue))
		g.FilesIndex.Mutex.RUnlock()


		dataReply := &packet.DataReply{
			Origin:      g.Name,
			Destination: dataRequest.Origin,
			HopLimit:    9,
			HashValue:   dataRequest.HashValue,
			Data:        chunk,}

		g.DSDV.Mutex.RLock()
		if g.DSDV.Contains(dataReply.Destination) {
			g.sendMessage(&packet.GossipPacket{DataReply: dataReply}, g.DSDV.NextHop[dataReply.Destination])
		}
		g.DSDV.Mutex.RUnlock()

	} else if dataRequest.HopLimit > 0 {
		dataRequest.HopLimit -= 1

		g.DSDV.Mutex.RLock()
		if g.DSDV.Contains(dataRequest.Destination) {
			g.sendMessage(&packet.GossipPacket{DataRequest: dataRequest}, g.DSDV.NextHop[dataRequest.Destination])
		}
		g.DSDV.Mutex.RUnlock()
	}
}

//SimpleMessageRoutine handle the GossipPackets of type SimpleMessage
//It first prints the message and g's Peers.
//Finally it forwards message to all g's Peers (except peerAddr)
func (g *Gossiper) SimpleMessageRoutine(message *packet.SimpleMessage, peerAddr *net.UDPAddr) {
	g.Peers.Mutex.RLock()
	defer g.Peers.Mutex.RUnlock()

	packet.PrintSimpleMessage(message)
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



	g.State.Mutex.Lock()
	if message.ID >= g.GetNextID(message.Origin) && message.Origin != g.Name {

		g.DSDV.Mutex.Lock()
		g.DSDV.Update(message, peerAddr)
		g.DSDV.Mutex.Unlock()


		g.State.UpdateGossiperState(message)
		g.sendStatusPacket(peerAddr)
		g.State.Mutex.Unlock()

		g.Rumormongering(message, false, peerAddr, nil)
	} else {

		g.sendStatusPacket(peerAddr)
		g.State.Mutex.Unlock()
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
	acked := g.AckRumors(peerAddr, statusPacket)
	g.pendingACK.Mutex.RUnlock()

	if !acked {
		g.StatusPacketHandler(statusPacket.Want, peerAddr, nil)
	} else {
		g.State.Mutex.RLock()
		//Check if S has messages that R has not seen yet
		peerVector := statusPacket.Want
		needsToSend, _ := g.HasOther(g.State.VectorClock, peerVector)

		//Check if R has messages that S has not seen yet
		wants, _ := g.HasOther(peerVector, g.State.VectorClock)
		g.State.Mutex.RUnlock()

		if !needsToSend && !wants {
			packet.PrintInSync(peerAddr)
		}
	}

}

func (g *Gossiper) StatusPacketHandler(peerVector []packet.PeerStatus, peerAddr *net.UDPAddr, rumorMessage *packet.RumorMessage) {

	g.State.Mutex.RLock()
	//Check if S has messages that R has not seen yet
	b, msg := g.HasOther(g.State.VectorClock, peerVector)

	if b {
		g.State.Mutex.RUnlock()
		g.Rumormongering(msg, false, nil, peerAddr)
		return
	}
	//Check if R has messages that S has not seen yet
	b, _ = g.HasOther(peerVector, g.State.VectorClock)

	if b {

		g.sendStatusPacket(peerAddr)
		g.State.Mutex.RUnlock()
		return
	}
	g.State.Mutex.RUnlock()

	if rumorMessage == nil {
		packet.PrintInSync(peerAddr)
	}

	if rand.Int()%2 == 0 && rumorMessage != nil {
		g.Rumormongering(rumorMessage, true, peerAddr, nil)
	}
}

//sendStatusPacket sends a StatusPacket to peerAddr that serves as an ACK to the RumorMessage.
func (g *Gossiper) sendStatusPacket(peerAddr *net.UDPAddr) {
	gossipPacket := &packet.GossipPacket{
		Status: &packet.StatusPacket{Want: g.State.VectorClock},
	}
	g.sendMessage(gossipPacket, peerAddr)
}

//PrivateMessageRoutine handles the private messages.
func (g *Gossiper) PrivateMessageRoutine(privateMessage *packet.PrivateMessage) {
	if privateMessage.Destination == g.Name {

		packet.PrintPrivateMessage(privateMessage)

		g.State.Mutex.Lock()
		g.State.UpdatePrivateQueue(privateMessage.Origin, privateMessage)
		g.State.Mutex.Unlock()

	} else if privateMessage.HopLimit > 0 {
		privateMessage.HopLimit -= 1

		g.DSDV.Mutex.RLock()
		if g.DSDV.Contains(privateMessage.Destination) {
			g.sendMessage(&packet.GossipPacket{Private: privateMessage}, g.DSDV.NextHop[privateMessage.Destination])
		}
		g.DSDV.Mutex.RUnlock()
	}
}
