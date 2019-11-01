package gossip

import (
	"fmt"
	"github.com/somecookie/Peerster/packet"
	"sync"
)

//GossiperState represents all the stateful information of the gossiper.
//It contains the following field:
//VectorClock: The list of all the next rumor messages per origin that the gossiper may receive
//ArchivedMessages: contains all messages received by all other peers
//RumorQueue: a queue of all rumor messages received. The purpose of this queue is to be sent to the GUI
//PrivateQueue: a map that maps each origin to a queue of all the private messages with that origin.
type GossiperState struct {
	VectorClock      []packet.PeerStatus
	ArchivedMessages map[string]map[uint32]packet.RumorMessage
	RumorQueue       []packet.RumorMessage
	PrivateQueue     map[string][]packet.PrivateMessage
	Mutex            sync.RWMutex
}

//GossiperStateFactory create a new empty state
func GossiperStateFactory() *GossiperState{
	return &GossiperState{
		VectorClock:      make([]packet.PeerStatus,0),
		ArchivedMessages: make(map[string]map[uint32]packet.RumorMessage),
		RumorQueue:       make([]packet.RumorMessage,0),
		PrivateQueue:     make(map[string][]packet.PrivateMessage),
		Mutex:            sync.RWMutex{},
	}

}

//UpdateGossiperState updates the vector clock and the archives.
//This method is thread-safe, no need to lock when using it.
func (gs *GossiperState) UpdateGossiperState(message *packet.RumorMessage) {
	gs.Mutex.Lock()
	gs.updateVectorClock(message)
	gs.Mutex.Unlock()

	gs.Mutex.Lock()
	gs.updateArchive(message)
	gs.Mutex.Unlock()
}

func (gs *GossiperState) updateVectorClock(message *packet.RumorMessage) {
	inVC := false
	for i, peerStat := range gs.VectorClock {
		if peerStat.Identifier == message.Origin {
			inVC = true
			if message.ID == peerStat.NextID {
				gs.VectorClock[i].NextID += 1
			}
			return
		}
	}

	if !inVC {
		var nextID uint32
		if message.ID == 1 {
			nextID = 2
		} else {
			nextID = 1
		}
		gs.VectorClock = append(gs.VectorClock, packet.PeerStatus{
			Identifier: message.Origin,
			NextID:     nextID,
		})
	}

}

func (gs *GossiperState) updateArchive(message *packet.RumorMessage) {
	if message.Text != "" {
		gs.RumorQueue = append(gs.RumorQueue, *message)
	}

	_, ok := gs.ArchivedMessages[message.Origin]

	if ok {
		gs.ArchivedMessages[message.Origin][message.ID] = *message
	} else {
		gs.ArchivedMessages[message.Origin] = make(map[uint32]packet.RumorMessage)
		gs.ArchivedMessages[message.Origin][message.ID] = *message
	}
}

func (gs *GossiperState) String() string{
	gs.Mutex.RLock()
	defer gs.Mutex.RUnlock()
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

	s += "======================================================================="

	return s
}

