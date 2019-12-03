package packet

import (
	"fmt"
	"net"
)

type RumorMessage struct {
	Origin string //message's original sender
	ID     uint32 //monotonically increasing sequence number assigned by the original sender
	Text   string //content of the message
}

func (rm *RumorMessage) String() string {
	return fmt.Sprintf("Origin: %s\nID: %d\nText: %s", rm.Origin, rm.ID, rm.Text)
}

func PrintRumorMessage(message *RumorMessage, peerAddr *net.UDPAddr) {
	//fmt.Printf("RUMOR origin %s from %s ID %d contents %s\n", message.Origin, peerAddr.String(), message.ID, message.Text)
}

func PrintMongering(addr *net.UDPAddr) {
	//fmt.Printf("MONGERING with %s\n", addr.String())
}

