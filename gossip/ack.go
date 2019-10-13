package gossip

import (
	"fmt"
	"github.com/somecookie/Peerster/packet"
	"net"
	"sync"
	"time"
)

const TIMEOUT = 10

type ACK struct {
	Origin       string
	ID           uint32
	AckedChannel chan bool
}

type PendingACK struct {
	ACKS  map[string][]ACK
	mutex sync.Mutex
}

func (pack *PendingACK) String() string{
	pack.mutex.Lock()
	s := ""
	for ip, acks := range pack.ACKS{
		s += ip+":\n"
		for _, ack := range acks{
			s += fmt.Sprintf("- Origin %s with ID %d\n", ack.Origin, ack.ID)
		}
	}
	pack.mutex.Unlock()
	return s
}

//WaitForAck waits for the acknowledgment. It timeouts after 10 seconds.
func (g *Gossiper) WaitForAck(message *packet.RumorMessage, peerAddr *net.UDPAddr) {
	ackedChannel := make(chan bool)
	ack := ACK{
		Origin:       message.Origin,
		ID:           message.ID,
		AckedChannel: ackedChannel,
	}
	g.AddToPendingACK(ack, peerAddr)

	ticker := time.NewTicker(TIMEOUT * time.Second)

	select {
		case <-ticker.C:
			g.Rumormongering(message, false)
		case <-ack.AckedChannel:
			g.RemoveACKed(ack, peerAddr)
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
	for i, a := range g.pendingACK.ACKS[peerAddr.String()]{
		if a == ack{
			g.pendingACK.ACKS[peerAddr.String()][i] =  g.pendingACK.ACKS[peerAddr.String()][length-1]
		}
	}
	g.pendingACK.ACKS[peerAddr.String()] =  g.pendingACK.ACKS[peerAddr.String()][:length-1]
	g.pendingACK.mutex.Unlock()
}

func (g *Gossiper) AckRumors(peerVector []packet.PeerStatus, peerAddr *net.UDPAddr) *packet.RumorMessage {
	g.pendingACK.mutex.Lock()
	maxID := uint32(0)
	var maxACK ACK
	for _, clock := range peerVector {
		for _, ack := range g.pendingACK.ACKS[peerAddr.String()] {
			if clock.Identifier == ack.Origin && ack.ID < clock.NextID {
				if ack.ID > maxID{
					maxID = ack.ID
					maxACK = ack
				}
				ack.AckedChannel <- true
			}
		}
	}
	g.pendingACK.mutex.Unlock()
	g.rumorState.Mutex.Lock()
	defer g.rumorState.Mutex.Unlock()
	return g.rumorState.ArchivedMessages[maxACK.Origin][maxACK.ID]
}
