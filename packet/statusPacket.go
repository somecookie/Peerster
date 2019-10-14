package packet

import (
	"fmt"
	"net"
	"sync"
)

type StatusPacket struct {
	Want []PeerStatus
}

type PeerStatus struct {
	Identifier string //origin's ID
	NextID     uint32 //lowest sequence number for which the peer has not yet seen a message from the origin
}

func (p PeerStatus) String() string{
	return fmt.Sprintf("Identifier %s with ID %d", p.Identifier, p.NextID)
}

type RumorState struct {
	VectorClock      []PeerStatus
	ArchivedMessages map[string]map[uint32]*RumorMessage
	Mutex            *sync.Mutex
}

func (r *RumorState) String() string{
	r.Mutex.Lock()
	s := "RumorState:\n"
	s += "VectorClock:\n"
	for _, vc := range r.VectorClock{
		s+= fmt.Sprintf("Origin: %s with nextID %d\n", vc.Identifier, vc.NextID)
	}
	for k1, v1 := range r.ArchivedMessages{
		s+= "From "+k1+"\n"
		for id, msg := range v1{
			s+= fmt.Sprintf("%d: %s\n", id, msg.Text)
		}
	}
	r.Mutex.Unlock()
	return s
}

func OutputStatusPacket(packet *StatusPacket, peerAddr *net.UDPAddr){
	s := fmt.Sprintf("STATUS from %s", peerAddr.String())
	for _, peerStatus := range packet.Want{
		//peer %s nextID %d
		s += fmt.Sprintf(" peer %s nextID %d", peerStatus.Identifier, peerStatus.NextID)
	}
	fmt.Println(s)
}

func OutputInSync(peerAddr *net.UDPAddr){
	fmt.Printf("IN SYNC WITH %s\n", peerAddr.String())
}

func OutputFlippedCoin(peerAddr *net.UDPAddr){
	fmt.Printf("FLIPPED COIN sending rumor to %s\n", peerAddr.String())
}