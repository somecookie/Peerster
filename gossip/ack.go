package gossip

import (
	"fmt"
	"github.com/somecookie/Peerster/packet"
	"math/rand"
	"net"
	"sync"
	"time"
)

const TIMEOUT = 10

type ACK struct {
	Origin       string
	ID           uint32
	AckedChannel chan *packet.StatusPacket
}

type PendingACK struct {
	ACKS  map[string][]ACK
	mutex sync.Mutex
}

func (pack *PendingACK) String() string {
	pack.mutex.Lock()
	s := ""
	for ip, acks := range pack.ACKS {
		s += ip + ":\n"
		for _, ack := range acks {
			s += fmt.Sprintf("- Origin %s with ID %d\n", ack.Origin, ack.ID)
		}
	}
	pack.mutex.Unlock()
	return s
}

//WaitForAck waits for the acknowledgment. It timeouts after 10 seconds.
func (g *Gossiper) WaitForAck(message *packet.RumorMessage, peerAddr *net.UDPAddr) {
	ackedChannel := make(chan *packet.StatusPacket)
	ack := ACK{
		Origin:       message.Origin,
		ID:           message.ID,
		AckedChannel: ackedChannel,
	}
	g.AddToPendingACK(ack, peerAddr)

	ticker := time.NewTicker(TIMEOUT * time.Second)
	defer ticker.Stop()

	select {
	case <-ticker.C:
		g.Rumormongering(message, false, nil, nil)
	case sp := <-ack.AckedChannel:
		g.RemoveACKed(ack, peerAddr)
		g.StatusPacketHandler(sp.Want, peerAddr, message)
	}
}

//AddToPendingACK adds the ack to the list of pending acknowledgment
func (g *Gossiper) AddToPendingACK(ack ACK, peerAddr *net.UDPAddr) {
	g.pendingACK.mutex.Lock()
	g.pendingACK.ACKS[peerAddr.String()] = append(g.pendingACK.ACKS[peerAddr.String()], ack)
	g.pendingACK.mutex.Unlock()
}

func (g *Gossiper) RemoveACKed(ack ACK, peerAddr *net.UDPAddr) {
	g.pendingACK.mutex.Lock()
	length := len(g.pendingACK.ACKS[peerAddr.String()])
	for i, a := range g.pendingACK.ACKS[peerAddr.String()] {
		if a == ack {
			g.pendingACK.ACKS[peerAddr.String()][i] = g.pendingACK.ACKS[peerAddr.String()][length-1]
		}
	}
	g.pendingACK.ACKS[peerAddr.String()] = g.pendingACK.ACKS[peerAddr.String()][:length-1]
	g.pendingACK.mutex.Unlock()
}

func (g *Gossiper) AckRumors(peerAddr *net.UDPAddr, statusPacket *packet.StatusPacket) bool {
	hasACKED := false
	peerVector := statusPacket.Want
	g.pendingACK.mutex.Lock()
	for _, clock := range peerVector {
		for _, ack := range g.pendingACK.ACKS[peerAddr.String()] {
			if clock.Identifier == ack.Origin && ack.ID < clock.NextID {
				hasACKED = true
				ack.AckedChannel <- statusPacket
			}
		}
	}
	g.pendingACK.mutex.Unlock()
	return hasACKED
}

func (g *Gossiper) StatusPacketHandler(peerVector []packet.PeerStatus, peerAddr *net.UDPAddr, rumorMessage *packet.RumorMessage) {
	//Check if S has messages that R has not seen yet
	g.rumorState.Mutex.Lock()

	b, msg := g.HasOther(g.rumorState.VectorClock, peerVector)

	if b {
		g.Rumormongering(msg, false, nil, peerAddr)
		g.rumorState.Mutex.Unlock()
		return
	}
	//Check if R has messages that S has not seen yet
	b, _ = g.HasOther(peerVector, g.rumorState.VectorClock)
	if b {
		g.sendStatusPacket(peerAddr)
		g.rumorState.Mutex.Unlock()
		return
	}
	g.rumorState.Mutex.Unlock()
	packet.OutputInSync(peerAddr)

	if rand.Int()%2 == 0 && rumorMessage != nil {
		g.Rumormongering(rumorMessage, true, peerAddr, nil)
	}
}

//HasOther checks if there are messages in thisVC that are not in thatVC.
func (g *Gossiper) HasOther(thisVC, thatVC []packet.PeerStatus) (bool, *packet.RumorMessage) {
	for _, sVC := range thisVC {
		inOtherVC := false
		for _, rVC := range thatVC {
			if sVC.Identifier == rVC.Identifier {
				inOtherVC = true
				if sVC.NextID > rVC.NextID {
					nextID := rVC.NextID
					origin := rVC.Identifier
					message := g.rumorState.ArchivedMessages[origin][nextID]
					return true, message

				}
			}
		}

		if !inOtherVC {
			nextID := uint32(1)
			origin := sVC.Identifier
			message := g.rumorState.ArchivedMessages[origin][nextID]
			return true, message
		}
	}
	return false, nil
}
