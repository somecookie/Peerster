package gossip

import (
	"github.com/somecookie/Peerster/packet"
	"sync"
)

//GossiperState represents all the stateful information of the gossiper.
//It contains the following field:
//VectorClock: The list of all the next rumor messages per origin that the gossiper may receive
//ArchivedMessages: contains all messages received by all other peers
//RumorQueue: a queue of all rumor messages received. The purpose of this queue is to be sent to the GUI
//PrivateQueue: maps each origin to a queue of all the private messages with that origin.
type GossiperState struct {
	VectorClock      []packet.PeerStatus
	ArchivedMessages map[string]map[uint32]packet.GossipPacket
	RumorQueue       []packet.GossipPacket
	PrivateQueue     map[string][]packet.PrivateMessage
	Mutex            sync.RWMutex
}

//GossiperStateFactory create a new empty state"
func GossiperStateFactory() *GossiperState{
	return &GossiperState{
		VectorClock:      make([]packet.PeerStatus,0),
		ArchivedMessages: make(map[string]map[uint32]packet.GossipPacket),
		RumorQueue:       make([]packet.GossipPacket,0),
		PrivateQueue:     make(map[string][]packet.PrivateMessage),
		Mutex:            sync.RWMutex{},
	}

}

//UpdateGossiperState updates the vector clock and the archives.
func (gs *GossiperState) UpdateGossiperState(gossipPacket *packet.GossipPacket) {
	var ID uint32
	var origin string

	if gossipPacket.Rumor != nil{
		ID = gossipPacket.Rumor.ID
		origin = gossipPacket.Rumor.Origin
	}else{
		ID = gossipPacket.TLCMessage.ID
		origin = gossipPacket.TLCMessage.Origin
	}

	gs.updateVectorClock(origin, ID)
	gs.updateArchive(origin, ID, gossipPacket)

}

//UpdatePrivateQueue enqueues the private message to the corresponding queue.
//The destination parameter is needed when you add you own message to the queue.
//This method is not thread-safe.
func (gs *GossiperState) UpdatePrivateQueue(destination string, privateMessage *packet.PrivateMessage){
	_, ok := gs.PrivateQueue[destination]

	if !ok{
		gs.PrivateQueue[destination] = make([]packet.PrivateMessage, 0, 1)
	}

	gs.PrivateQueue[destination] = append(gs.PrivateQueue[destination], *privateMessage)

}

func (gs *GossiperState) updateVectorClock(origin string, id uint32) {
	inVC := false
	for i, peerStat := range gs.VectorClock {
		if peerStat.Identifier == origin{
			inVC = true
			if id == peerStat.NextID {
				gs.VectorClock[i].NextID += 1
			}
			return
		}
	}

	if !inVC {
		var nextID uint32
		if id == 1 {
			nextID = 2
		} else {
			nextID = 1
		}
		gs.VectorClock = append(gs.VectorClock, packet.PeerStatus{
			Identifier: origin,
			NextID:     nextID,
		})
	}

}


func (gs *GossiperState) updateArchive(origin string, id uint32, message *packet.GossipPacket) {

	_, ok := gs.ArchivedMessages[origin]

	if !ok {
		gs.ArchivedMessages[origin] = make(map[uint32]packet.GossipPacket)
	}

	_, ok = gs.ArchivedMessages[origin][id]

	if !ok {
		gs.ArchivedMessages[origin][id] = *message

		if (message.Rumor != nil && message.Rumor.Text != "") || (message.TLCMessage != nil && message.TLCMessage.Confirmed != -1){
			gs.RumorQueue = append(gs.RumorQueue, *message)
		}
	}



}

/*func (gs *GossiperState) String() string{
	s := "=======================================================================\n"
	s += "============================Vector Clock===============================\n"

	for _, clock := range gs.VectorClock{
		s += fmt.Sprintf("%s %d\n", clock.Identifier, clock.NextID)
	}

	s += "=======================================================================\n"
	s += "==============================Archives=================================\n"
	for origin, rumors := range gs.ArchivedMessages{
		s+= fmt.Sprintf("From %s\n", origin)
		for i, msg := range rumors{
			s += fmt.Sprintf("%d: %s\n",i, msg.Text)
		}
	}

	s += "=======================================================================\n"
	s += "=============================RumorQueue================================\n"
	for _, msg:= range gs.RumorQueue{
		s += fmt.Sprintf("From %s(%d): %s\n", msg.Origin, msg.ID, msg.Text)
	}

	s += "=======================================================================\n"
	s += "===========================PrivateQueue================================\n"

	for dest, queue := range gs.PrivateQueue{
		s += fmt.Sprintf("Conversation with %s\n", dest)
		for _, msg := range queue{
			s += fmt.Sprintf("%s: %s\n", msg.Origin, msg.Text)
		}
	}

	s += "======================================================================="

	return s
}*/

