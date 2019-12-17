package gossip

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/somecookie/Peerster/helper"
	"github.com/somecookie/Peerster/packet"
	"math/rand"
	"net"
	"time"
)

var hasher = sha256.New()

func (g *Gossiper) GossipPacketHandler(receivedPacket *packet.GossipPacket, from *net.UDPAddr) {
	if g.simple {
		if receivedPacket.Simple != nil {
			go g.SimpleMessageRoutine(receivedPacket.Simple, from)
		}
	} else {
		if receivedPacket.Rumor != nil || receivedPacket.TLCMessage != nil{
			go g.RumorMessageRoutine(receivedPacket, from)
		} else if receivedPacket.Status != nil {
			go g.StatusPacketRoutine(receivedPacket.Status, from)
		} else if receivedPacket.Private != nil {
			go g.PrivateMessageRoutine(receivedPacket.Private)
		} else if receivedPacket.DataRequest != nil {
			go g.DataRequestRoutine(receivedPacket.DataRequest)
		} else if receivedPacket.DataReply != nil {
			go g.DataReplyRoutine(receivedPacket.DataReply)
		} else if receivedPacket.SearchRequest != nil {
			g.SearchRequestRoutine(receivedPacket.SearchRequest, from)
		} else if receivedPacket.SearchReply != nil {
			g.SearchReplyRoutine(receivedPacket.SearchReply)
		}else if receivedPacket.Ack != nil{
			go g.TLCAckRoutine(receivedPacket.Ack)
		}
	}

}

//DataReplyRoutine handles the incoming dataReply.
func (g *Gossiper) DataReplyRoutine(dataReply *packet.DataReply) {
	if dataReply.Destination == g.Name {

		hasher.Reset()
		_, err := hasher.Write(dataReply.Data)

		if err != nil {
			helper.LogError(err)
			return
		}
		hash := hasher.Sum(nil)
		receivedHashString := hex.EncodeToString(dataReply.HashValue)
		g.Requested.Mutex.RLock()
		if hex.EncodeToString(hash) == receivedHashString || len(dataReply.Data) == 0 || dataReply.Data == nil {
			if origins, ok := g.Requested.ACKs[dataReply.Origin]; ok {
				if c, ok := origins[receivedHashString]; ok {
					c <- dataReply
				}
			}
		}
		g.Requested.Mutex.RUnlock()
	} else if dataReply.HopLimit > 0 {
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
func (g *Gossiper) RumorMessageRoutine(gossipPacket *packet.GossipPacket, peerAddr *net.UDPAddr) {

	origin,ID := gossipPacket.GetOriginAndID()

	if ID == 0 && origin == ""{
		return
	}

	var text string
	if gossipPacket.Rumor != nil{
		packet.PrintRumorMessage(gossipPacket.Rumor, peerAddr)
		g.Peers.Mutex.RLock()
		PrintPeers(g)
		g.Peers.Mutex.RUnlock()
		text = gossipPacket.Rumor.Text
	}else{
		text = ""
	}


	g.State.Mutex.Lock()
	if ID >= g.GetNextID(origin) && origin!= g.Name {

		g.DSDV.Mutex.Lock()
		g.DSDV.Update(ID, origin,text, peerAddr)
		g.DSDV.Mutex.Unlock()

		g.State.UpdateGossiperState(gossipPacket)
		g.sendStatusPacket(peerAddr)

		g.Rumormongering(gossipPacket, false, peerAddr, nil)
	} else {

		g.sendStatusPacket(peerAddr)
	}
	g.State.Mutex.Unlock()

	if gossipPacket.TLCMessage != nil && origin != g.Name{
		g.HandleTLCMessage(gossipPacket.TLCMessage)
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

func (g *Gossiper) StatusPacketHandler(peerVector []packet.PeerStatus, peerAddr *net.UDPAddr, gossipPacket *packet.GossipPacket) {

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

	if gossipPacket == nil {
		packet.PrintInSync(peerAddr)
	}

	if rand.Int()%2 == 0 && gossipPacket != nil {
		//log.Println("Mongering flipped coin")
		g.Rumormongering(gossipPacket, true, peerAddr, nil)
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

//SearchRequestRoutine is the routine that handles the SearchRequest
//sr *packet.SearchRequest is the search we have to handle
//from *net.UDPAddr is the address of the node from whom we received the SearchRequest. If from is nil, this means
//that the SearchRequest comes from the client.
func (g *Gossiper) SearchRequestRoutine(sr *packet.SearchRequest, from *net.UDPAddr) {
	if !g.DSR.Contains(sr) {
		g.DSR.Add(sr)

		if sr.Origin != g.Name {
			g.FilesIndex.Mutex.RLock()
			results := g.FilesIndex.FindMatchingFiles(sr.Keywords)
			g.FilesIndex.Mutex.RUnlock()
			g.DSDV.Mutex.RLock()
			if len(results) > 0 && g.DSDV.Contains(sr.Origin) {
				sreply := &packet.SearchReply{
					Origin:      g.Name,
					Destination: sr.Origin,
					HopLimit:    9,
					Results:     results,
				}

				g.sendMessage(&packet.GossipPacket{SearchReply: sreply}, g.DSDV.NextHop[sr.Origin])
			}
			g.DSDV.Mutex.RUnlock()

		}

		g.Peers.Mutex.RLock()
		g.redistributeBudget(sr, from)
		g.Peers.Mutex.RUnlock()

		go g.startDuplicateTimer(sr)

	}

}

//startDuplicateTimer starts a timer of 0.5 seconds and then remove sr from the set of possible duplicate search request
func (g *Gossiper) startDuplicateTimer(sr *packet.SearchRequest) {

	ticker := time.NewTicker(500 * time.Millisecond)

	select {
	case <-ticker.C:
		g.DSR.Remove(sr)
	}

}

//redistributeBudget redistributes the budget of a given SearchRequest sr by forwarding
//it to other peers with an evenly distributed budget based on the sr's budget.
//sr *packet.SearchRequest is the search request we need to forward with a redistributed budget.
//from *net.UDPAddr is the address of the node from whom we received the SearchRequest. If from is nil, this means
//that the SearchRequest comes from the client.
func (g *Gossiper) redistributeBudget(sr *packet.SearchRequest, from *net.UDPAddr) {
	remainingBudget := sr.Budget - 1
	nbrPeers := uint64(len(g.Peers.Set))
	if from != nil {
		//remainingBudget -= 1
		nbrPeers -= 1
	}

	if nbrPeers > 0 && remainingBudget > 0 && remainingBudget < nbrPeers {

		var randomPeers []*net.UDPAddr
		if from == nil {
			randomPeers = g.Peers.NRandom(remainingBudget)
		} else {
			randomPeers = g.Peers.NRandom(remainingBudget, from)
		}

		forwardSR := &packet.SearchRequest{
			Origin:   sr.Origin,
			Budget:   1,
			Keywords: sr.Keywords,
		}

		for _, dest := range randomPeers {
			g.sendMessage(&packet.GossipPacket{SearchRequest: forwardSR}, dest)
		}
	} else if nbrPeers > 0 && remainingBudget > 0 {

		newBudget := remainingBudget / nbrPeers
		exceeding := remainingBudget % nbrPeers

		i := uint64(0)
		for _, dest := range g.Peers.Set {

			if from == nil || from.String() != dest.String() {
				forwardSR := &packet.SearchRequest{
					Origin:   sr.Origin,
					Budget:   newBudget,
					Keywords: sr.Keywords,
				}

				if i < exceeding {
					i++
					forwardSR.Budget += 1
				}

				g.sendMessage(&packet.GossipPacket{SearchRequest: forwardSR}, dest)
			}

		}

	}
}

func (g *Gossiper) SearchReplyRoutine(reply *packet.SearchReply) {
	g.fullMatches.Lock()
	if reply.Destination == g.Name{
		if  g.fullMatches.n < THRESHOLD_MATCHES {
			for _, result := range reply.Results {

				if len(result.ChunkMap) == 0{
					continue
				}

				newResult := g.Matches.AddNewResult(result, reply.Origin)

				if newResult {
					packet.PrintSearchResult(result, reply.Origin)
				}

				if newResult && result.ChunkCount == uint64(len(result.ChunkMap)) {
					g.Matches.Lock()
					g.Matches.Queue = append(g.Matches.Queue, struct {
						FileName string
						Origin   string
						MetaHash string
					}{FileName: result.FileName, Origin: reply.Origin, MetaHash:hex.EncodeToString(result.MetafileHash)})
					g.Matches.Unlock()

					g.fullMatches.n += 1
					if g.fullMatches.n == THRESHOLD_MATCHES {
						fmt.Println("SEARCH FINISHED")
						g.fullMatches.Unlock()
						return
					}
				}
			}
		}
		g.fullMatches.Unlock()
	} else if reply.HopLimit > 0 {
		g.fullMatches.Unlock()
		g.DSDV.Mutex.RLock()
		defer g.DSDV.Mutex.RUnlock()
		reply.HopLimit -= 1
		if g.DSDV.Contains(reply.Destination) {
			g.sendMessage(&packet.GossipPacket{SearchReply: reply}, g.DSDV.NextHop[reply.Destination])
		}
	}
}
