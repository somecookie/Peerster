package packet

import (
	"fmt"
	"net"
)

type StatusPacket struct {
	Want []PeerStatus
}

type PeerStatus struct {
	Identifier string //origin's ID
	NextID     uint32 //lowest sequence number for which the peer has not yet seen a message from the origin
}

func (p PeerStatus) String() string {
	return fmt.Sprintf("Identifier %s with ID %d", p.Identifier, p.NextID)
}

func PrintStatusPacket(packet *StatusPacket, peerAddr *net.UDPAddr) {
	s := fmt.Sprintf("STATUS from %s", peerAddr.String())
	for _, peerStatus := range packet.Want {
		//peer %s nextID %d
		s += fmt.Sprintf(" peer %s nextID %d", peerStatus.Identifier, peerStatus.NextID)
	}
	//fmt.Println(s)
}

func PrintInSync(peerAddr *net.UDPAddr) {
	//fmt.Printf("IN SYNC WITH %s\n", peerAddr.String())
}

func PrintFlippedCoin(peerAddr *net.UDPAddr) {
	//fmt.Printf("FLIPPED COIN sending rumor to %s\n", peerAddr.String())
}
