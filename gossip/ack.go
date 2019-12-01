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
}

type PendingACK struct {
	ACKS  map[string]map[ACK]chan *packet.StatusPacket
	Mutex sync.RWMutex
}


//WaitForAck waits for the acknowledgment. It timeouts after 10 seconds.
func (g *Gossiper) WaitForAck(message *packet.GossipPacket, peerAddr *net.UDPAddr) {

	origin, id := message.GetOriginAndID()

	g.pendingACK.Mutex.Lock()
	ackChannel:=g.AddToPendingACK(origin, id ,peerAddr)
	g.pendingACK.Mutex.Unlock()

	if ackChannel == nil{
		return
	}

	ticker := time.NewTicker(TIMEOUT * time.Second)
	defer ticker.Stop()

	select {
	case <-ticker.C:

		g.pendingACK.Mutex.Lock()
		g.RemoveACKed(message, peerAddr)
		g.pendingACK.Mutex.Unlock()
		g.Rumormongering(message, false, nil, nil)

	case sp := <-ackChannel:
		g.pendingACK.Mutex.Lock()
		g.RemoveACKed(message, peerAddr)
		g.pendingACK.Mutex.Unlock()

		g.StatusPacketHandler(sp.Want, peerAddr, message)
	}
}

//AddToPendingACK adds the ack to the List of pending acknowledgment
func (g *Gossiper) AddToPendingACK(origin string, ID uint32, peerAddr *net.UDPAddr) chan *packet.StatusPacket {
	new := false
	ack := ACK{
		Origin: origin,
		ID:     ID,
	}
	if _, ok := g.pendingACK.ACKS[peerAddr.String()]; !ok{
		g.pendingACK.ACKS[peerAddr.String()] = make(map[ACK]chan *packet.StatusPacket)
		new = true
	}else{
		if _, ok := g.pendingACK.ACKS[peerAddr.String()][ack]; !ok{
			new = true
		}
	}

	if new{
		ackChannel := make(chan *packet.StatusPacket, 100) //arbitrary value, should be tested..
		g.pendingACK.ACKS[peerAddr.String()][ack] = ackChannel
		return ackChannel
	}else{
		return nil
	}


}

func (g *Gossiper) RemoveACKed(message *packet.GossipPacket, peerAddr *net.UDPAddr) {
	origin, id := message.GetOriginAndID()
	ack := ACK{
		Origin: origin,
		ID:     id,
	}

	if c, ok:= g.pendingACK.ACKS[peerAddr.String()][ack]; ok{
		close(c)
		delete(g.pendingACK.ACKS[peerAddr.String()], ack)
	}

}

func (g *Gossiper) AckRumors(peerAddr *net.UDPAddr, statusPacket *packet.StatusPacket) bool {
	hasACKED := false
	peerVector := statusPacket.Want

	for _, clock := range peerVector {
		for ack,c := range g.pendingACK.ACKS[peerAddr.String()] {
			if clock.Identifier == ack.Origin && ack.ID < clock.NextID {
				hasACKED = true
				c <- statusPacket
			}
		}
	}
	return hasACKED
}

//HasOther checks if there are messages in thisVC that are not in thatVC.
func (g *Gossiper) HasOther(thisVC, thatVC []packet.PeerStatus) (bool, *packet.GossipPacket) {
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
