package gossip

import (
	"github.com/somecookie/Peerster/packet"
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
	ACKS  map[string]map[ACK]bool
	Mutex sync.RWMutex
}

//WaitForAck waits for the acknowledgment. It timeouts after 10 seconds.
func (g *Gossiper) WaitForAck(message *packet.RumorMessage, peerAddr *net.UDPAddr) {
	ackedChannel := make(chan *packet.StatusPacket)
	ack := ACK{
		Origin:       message.Origin,
		ID:           message.ID,
		AckedChannel: ackedChannel,
	}

	g.pendingACK.Mutex.Lock()
	g.AddToPendingACK(ack, peerAddr)
	g.pendingACK.Mutex.Unlock()

	ticker := time.NewTicker(TIMEOUT * time.Second)
	defer ticker.Stop()

	select {
	case <-ticker.C:

		g.pendingACK.Mutex.Lock()
		g.RemoveACKed(ack, peerAddr)
		g.pendingACK.Mutex.Unlock()

		g.Rumormongering(message, false, nil, nil)

	case sp := <-ack.AckedChannel:
		g.pendingACK.Mutex.Lock()
		g.RemoveACKed(ack, peerAddr)
		g.pendingACK.Mutex.Unlock()

		g.StatusPacketHandler(sp.Want, peerAddr, message)
	}
}

//AddToPendingACK adds the ack to the List of pending acknowledgment
func (g *Gossiper) AddToPendingACK(ack ACK, peerAddr *net.UDPAddr) {
	if _, ok := g.pendingACK.ACKS[peerAddr.String()]; !ok{
		g.pendingACK.ACKS[peerAddr.String()] = make(map[ACK]bool)
	}
	g.pendingACK.ACKS[peerAddr.String()][ack] = true
}

func (g *Gossiper) RemoveACKed(ack ACK, peerAddr *net.UDPAddr) {
	delete(g.pendingACK.ACKS[peerAddr.String()], ack)
}

func (g *Gossiper) AckRumors(peerAddr *net.UDPAddr, statusPacket *packet.StatusPacket) bool {
	hasACKED := false
	peerVector := statusPacket.Want

	for _, clock := range peerVector {
		for ack := range g.pendingACK.ACKS[peerAddr.String()] {
			if clock.Identifier == ack.Origin && ack.ID < clock.NextID {
				hasACKED = true
				ack.AckedChannel <- statusPacket
			}
		}
	}
	return hasACKED
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
					message := g.State.ArchivedMessages[origin][nextID]
					return true, &message

				}
			}
		}

		if !inOtherVC {
			nextID := uint32(1)
			origin := sVC.Identifier
			message := g.State.ArchivedMessages[origin][nextID]
			return true, &message
		}
	}
	return false, nil
}
