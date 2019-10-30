package packet

import "fmt"

//PrivateMessage represents the messages sent privately by the nodes using their names.
//Origin (string) contains the identifier of the node sending the packet.
//ID (uint32) is always set to which denotes no sequencing.
//Text (string) contains the text of the private message
//Destination (string) is the destination node identifier of the message
//HopLimit (uint32) denotes the number of nodes that can be reached before the message is discarded
type PrivateMessage struct {
	Origin      string
	ID          uint32
	Text        string
	Destination string
	HopLimit    uint32
}

var MaxHops uint32 = 10

func PrintPrivateMessage(pm *PrivateMessage){
	fmt.Printf("PRIVATE origin %s hop-limit %d contents %s\n", pm.Origin, pm.HopLimit, pm.Text)
}
