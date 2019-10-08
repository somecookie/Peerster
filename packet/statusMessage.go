package packet

type StatusPacket struct {
	Want []PeerStatus
}

type PeerStatus struct {
	Identifier string //origin's ID
	NextID     uint32 //lowest sequence number for which the peer has not yet seen a message from the origin
}
