package gossip

import (
	"github.com/somecookie/Peerster/packet"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type TLCMajority struct {
	sync.RWMutex
	Acks         map[uint32][]string
	AcksChannels map[uint32]chan bool
	Majority     int
}

func TLCMajortiyFactory(N int) *TLCMajority {
	return &TLCMajority{
		RWMutex:  sync.RWMutex{},
		Acks:     make(map[uint32][]string),
		Majority: N,
	}
}

func (tm *TLCMajority) SelfAdd(ID uint32, selfName string) {
	tm.Lock()
	defer tm.Unlock()

	tm.Acks[ID] = make([]string, 0, tm.Majority/2+1)
	tm.Acks[ID] = append(tm.Acks[ID], selfName)
	tm.AcksChannels[ID] = make(chan bool)
}

func (tm *TLCMajority) AddNewAck(ack *packet.TLCAck) bool {
	tm.Lock()
	defer tm.Unlock()

	if len(tm.Acks) > tm.Majority/2 {
		return false
	}

	if acks, ok := tm.Acks[ack.ID]; ok {
		contains := false
		for _, name := range acks {
			if name == ack.Origin {
				contains = true
			}
		}

		if !contains {
			tm.Acks[ack.ID] = append(tm.Acks[ack.ID], ack.Origin)
		}

		if len(tm.Acks) > tm.Majority/2 {
			c := tm.AcksChannels[ack.ID]
			c <- true
			return true
		}
	}

	return false
}

func (g *Gossiper) TLCRoutine(gossipPacket *packet.GossipPacket, from *net.UDPAddr) {
	tlcMessage := gossipPacket.TLCMessage

	g.State.Mutex.RLock()
	if tlcMessage.ID >= g.GetNextID(tlcMessage.Origin){
		//new message => update state and store it
		g.DSDV.Mutex.Lock()
		g.DSDV.Update(tlcMessage.ID, tlcMessage.Origin,"", from)
		g.DSDV.Mutex.Unlock()

		g.State.UpdateGossiperState(gossipPacket)
		g.sendStatusPacket(from)



	}


}

func (g *Gossiper) AckRoutine(ack *packet.TLCAck) {
	g.TLCMajority.AddNewAck(ack)
}

func (g *Gossiper) Stubborn(tlcMessage *packet.TLCMessage) {
	ticker := time.NewTicker(time.Duration(g.stubbornTimeout) * time.Second)
	defer ticker.Stop()

	g.TLCMajority.RLock()
	ackChannel := g.TLCMajority.AcksChannels[tlcMessage.ID]
	g.TLCMajority.RUnlock()

	for {
		select {
		case <-ticker.C:
			g.Rumormongering(&packet.GossipPacket{TLCMessage: tlcMessage}, false, nil, nil)
		case <-ackChannel:
			close(ackChannel)
			atomic.AddUint32(&g.counter, 1)
			confirmation := &packet.TLCMessage{
				Origin:      g.Name,
				ID:          g.counter,
				Confirmed:   int(tlcMessage.ID),
				TxBlock:     tlcMessage.TxBlock,
				VectorClock: tlcMessage.VectorClock,
				Fitness:     tlcMessage.Fitness,
			}

			g.Rumormongering(&packet.GossipPacket{TLCMessage:confirmation}, false, nil, nil)
			return
		}
	}
}
