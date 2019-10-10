package packet

type PeerStatus struct {
	Identifier string //origin's ID
	NextID     uint32 //lowest sequence number for which the peer has not yet seen a message from the origin
}

type RumorState struct {
	VectorClock      []PeerStatus
	ArchivedMessages map[string]map[uint32]*RumorMessage
}
