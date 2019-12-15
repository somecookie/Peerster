package gossip

import (
	"fmt"
	"github.com/somecookie/Peerster/blockchain"
	"github.com/somecookie/Peerster/fileSharing"
	"github.com/somecookie/Peerster/packet"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

//TLCMajority represents the state for the majorities. All operations on this structure are thread safe.
//Acks         map[uint32][]string is the mapping between an ID and all the peers that have already acknowledged the TLCMessage with this ID
//AcksChannels map[uint32]chan bool is the mapping an ID of a TLCMessage and its associated channel
//Total        int is the total number of Peers in the network. Therefore, the majority is Total/2.
//MyRound      uint32
//Queue        *fileSharing.MetadataQueue is the FIFO used to get the indexing command.
//Confirmed    map[uint32][]string  is a mapping from a round to the gossiper who sent a confirmed message.
//FutureMsg    []packet.TLCMessage are the messages that were not accepted because of their vector clock.
//LastID       uint32 is the last ID used by the gossiper owning the TLCMajority
type TLCMajority struct {
	sync.RWMutex
	Acks         map[uint32][]string
	AcksChannels map[uint32]chan bool
	Total        int
	MyRound      uint32
	OtherRounds  map[string]uint32
	Queue        *fileSharing.MetadataQueue
	Confirmed    map[uint32][]Confirmation
	FutureMsg    []*packet.TLCMessage
	LastID       uint32
	ReicvCommand bool
}

type Confirmation struct {
	Origin string
	ID     uint32
}

func TLCMajorityFactory(N int) *TLCMajority {
	return &TLCMajority{
		RWMutex:      sync.RWMutex{},
		Acks:         make(map[uint32][]string),
		AcksChannels: make(map[uint32]chan bool),
		Total:        N,
		MyRound:      0,
		OtherRounds:  make(map[string]uint32),
		Queue:        fileSharing.MetadataQueueFactory(),
		Confirmed:    make(map[uint32][]Confirmation),
		FutureMsg:    make([]*packet.TLCMessage, 0),
		LastID:       0,
		ReicvCommand: false,
	}
}

//SelfAdd allows the gossiper to add itself when it creates a new TLCMessage.
//ID uint32 is the ID of the newly created TLCMessage
//selfName string is the name of the gossiper
func (tm *TLCMajority) SelfAdd(ID uint32, selfName string) {

	tm.Acks[ID] = make([]string, 0, tm.Total/2+1)
	tm.Acks[ID] = append(tm.Acks[ID], selfName)
	tm.AcksChannels[ID] = make(chan bool)
}

//AddNewAck adds a new ack to the list of peers that acknowledged the TLCMessage being acknowledged by ack.
//ack *packet.TLCAck is the new ack
//returns a boolean that tells if the majority was reached (for the first time).
//Notice that the majority can only be reached by one ack, the next ack will simply be discarded.
//If the majority, the channel associated to the ack allows to stop the stubborn timer and continue the process.
func (tm *TLCMajority) AddNewAck(ack *packet.TLCAck) bool {

	if len(tm.Acks[ack.ID]) > tm.Total/2 {
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

		if len(tm.Acks[ack.ID]) > tm.Total/2 {
			c := tm.AcksChannels[ack.ID]
			c <- true
			return true
		}
	}

	return false
}

func (tm *TLCMajority) PrintReBroadcast(confirmed *packet.TLCMessage) {
	peersList := tm.Acks[uint32(confirmed.Confirmed)]
	witnesses := strings.Join(peersList, ",")
	fmt.Printf("RE-BROADCAST ID %d WITNESSES %s\n", confirmed.Confirmed, witnesses)
}

//GetChannel allows you to get the channel corresponding to the given id. It returns nil if the ID is not valid.
func (tm *TLCMajority) GetChannel(ID uint32) chan bool {
	if c, ok := tm.AcksChannels[ID]; ok {
		return c
	} else {
		return nil
	}
}

func (tm *TLCMajority) RemoveFromFuture(message *packet.TLCMessage) {
	for i, msg := range tm.FutureMsg {
		if message == msg {
			tm.FutureMsg[i], tm.FutureMsg[len(tm.FutureMsg)-1] = tm.FutureMsg[len(tm.FutureMsg)-1], tm.FutureMsg[i]
			tm.FutureMsg = tm.FutureMsg[:len(tm.FutureMsg)-1]
			return
		}
	}
}

//Stubborn stubbornly rebroadcast the tlcMessage after a timeout or, if a majority is reached before the timeout
//re-broadcasts the new confirmed message.
//tlcMessage *packet.TLCMessage is the new TLC message broadcast in the first place.
func (g *Gossiper) Stubborn(tlcMessage *packet.TLCMessage) {
	ticker := time.NewTicker(time.Duration(g.stubbornTimeout) * time.Second)

	g.TLCMajority.RLock()
	ackChannel := g.TLCMajority.GetChannel(tlcMessage.ID)
	g.TLCMajority.RUnlock()

	if ackChannel == nil {
		return
	}

	for {
		select {
		case <-ticker.C:
			g.Rumormongering(&packet.GossipPacket{TLCMessage: tlcMessage}, false, nil, nil)
		case majority := <-ackChannel:
			ticker.Stop()
			close(ackChannel)
			if majority {
				atomic.AddUint32(&g.counter, 1)
				confirmation := &packet.TLCMessage{
					Origin:      g.Name,
					ID:          g.counter,
					Confirmed:   int(tlcMessage.ID),
					TxBlock:     tlcMessage.TxBlock,
					VectorClock: tlcMessage.VectorClock,
					Fitness:     tlcMessage.Fitness,
				}
				g.TLCMajority.RLock()
				g.TLCMajority.PrintReBroadcast(confirmation)
				g.TLCMajority.RUnlock()

				gp := &packet.GossipPacket{TLCMessage: confirmation}
				g.State.UpdateGossiperState(gp)
				g.Rumormongering(gp, false, nil, nil)

				if !g.ackAll{
					g.TLCMajority.Lock()
					g.TLCMajority.Confirmed[g.TLCMajority.MyRound] = append(g.TLCMajority.Confirmed[g.TLCMajority.MyRound], Confirmation{
						Origin: g.Name,
						ID:     confirmation.ID,
					})
					g.TryNextRound()
					g.TLCMajority.Unlock()
				}


			}
			return
		}
	}
}

//BroadcastNewFile gossips a newly indexed file
//metadata *fileSharing.Metadata is the metadata of the new file
func (g *Gossiper) BroadcastNewFile(metadata *fileSharing.Metadata) {
	bp := blockchain.BlockPublish{
		PrevHash: [32]byte{},
		Transaction: blockchain.TxPublish{
			Name:         metadata.Name,
			Size:         int64(metadata.Size),
			MetafileHash: metadata.MetaHash,
		},
	}

	g.State.Mutex.RLock()
	vc := &packet.StatusPacket{
		Want: g.State.VectorClock,
	}
	g.State.Mutex.RUnlock()

	atomic.AddUint32(&g.counter, 1)
	g.TLCMajority.LastID = g.counter
	tlcMsg := &packet.TLCMessage{
		Origin:      g.Name,
		ID:          g.counter,
		Confirmed:   -1,
		TxBlock:     bp,
		VectorClock: vc,
		Fitness:     0,
	}
	gp := &packet.GossipPacket{TLCMessage: tlcMsg}
	g.TLCMajority.SelfAdd(tlcMsg.ID, g.Name)
	g.State.UpdateGossiperState(gp)

	packet.PrintUnconfirmedMessage(tlcMsg)
	g.Rumormongering(gp, false, nil, nil)
	go g.Stubborn(tlcMsg)
}

//This function handles the TLCMessage when they are received by the gossiper
//tlcMessage *packet.TLCMessage is the TLCMessage received by the gossiper
func (g *Gossiper) HandleTLCMessage(tlcMessage *packet.TLCMessage) {

	if g.ackAll{
		if tlcMessage.Confirmed == -1 {
			packet.PrintUnconfirmedMessage(tlcMessage)

			ack := &packet.TLCAck{
				Origin:      g.Name,
				ID:          tlcMessage.ID,
				Destination: tlcMessage.Origin,
				HopLimit:    uint32(g.hoplimit),
			}
			packet.PrintSendingTLCAck(ack)
			g.TLCAckRoutine(ack)

		} else {
			packet.PrintConfirmedMessage(tlcMessage)
		}
		return
	}

	//check validity
	g.TLCMajority.Lock()
	defer g.TLCMajority.Unlock()

	g.TLCMajority.FutureMsg = append(g.TLCMajority.FutureMsg, tlcMessage)

	for _, msg := range g.TLCMajority.FutureMsg {
		if g.SatisfyVC(msg) {
			g.TLCMajority.RemoveFromFuture(msg)
			if msg.Confirmed == -1 {
				packet.PrintUnconfirmedMessage(tlcMessage)

				if _, ok := g.TLCMajority.OtherRounds[msg.Origin]; !ok {
					g.TLCMajority.OtherRounds[msg.Origin] = 1
				} else {
					g.TLCMajority.OtherRounds[msg.Origin] += 1
				}

				if round, ok := g.TLCMajority.OtherRounds[tlcMessage.Origin]; !ok {
					if g.TLCMajority.MyRound != 0 {
						return
					}
				} else if round-1 < g.TLCMajority.MyRound {
					return
				}

				ack := &packet.TLCAck{
					Origin:      g.Name,
					ID:          msg.ID,
					Destination: msg.Origin,
					HopLimit:    uint32(g.hoplimit),
				}

				packet.PrintSendingTLCAck(ack)
				g.TLCAckRoutine(ack)

			} else {
				packet.PrintConfirmedMessage(msg)
				r := g.TLCMajority.OtherRounds[msg.Origin] - 1
				g.TLCMajority.Confirmed[r] = append(g.TLCMajority.Confirmed[r], Confirmation{
					Origin: msg.Origin,
					ID:     msg.ID,
				})
				g.TryNextRound()
			}
		}
	}

}

func (g *Gossiper) SatisfyVC(msg *packet.TLCMessage) bool {
	g.State.Mutex.RLock()
	defer g.State.Mutex.RUnlock()

	for _, ps := range g.State.VectorClock {
		for _, psMsg := range msg.VectorClock.Want {
			if ps.Identifier == psMsg.Identifier && ps.NextID < psMsg.NextID {
				return false
			}
		}
	}

	for _, psMsg := range msg.VectorClock.Want {
		inVC := false
		for _, ps := range g.State.VectorClock {
			if psMsg.Identifier == ps.Identifier {
				inVC = true
			}
		}

		if !inVC {
			return false
		}
	}

	return true
}

//TLCAckRoutine handles the incoming acks, either by updating the majority counter if
//the gossiper is the destination or by forwarding the ack to the next hop
//ack *packet.TLCAck is the received ack
func (g *Gossiper) TLCAckRoutine(ack *packet.TLCAck) {
	if ack.Destination == g.Name {
		g.TLCMajority.AddNewAck(ack)
	} else if ack.HopLimit > 0 {
		ack.HopLimit -= 1

		g.DSDV.Mutex.RLock()
		if g.DSDV.Contains(ack.Destination) {
			nextHop := g.DSDV.NextHop[ack.Destination]
			gossipPacket := &packet.GossipPacket{Ack: ack}
			g.sendMessage(gossipPacket, nextHop)
		}
		g.DSDV.Mutex.RUnlock()

	}
}

func (g *Gossiper) TryNextRound() {
	tm := g.TLCMajority
	confirmed := tm.Confirmed[tm.MyRound]
	if len(confirmed) > tm.Total/2 && tm.ReicvCommand {

		tm.MyRound += 1
		g.PrintNextRound(tm.MyRound, confirmed)



		if len(tm.Acks[tm.LastID]) <= tm.Total/2 {
			tm.AcksChannels[tm.LastID] <- false
		}

		if metdata := tm.Queue.Dequeue(); metdata != nil {
			g.BroadcastNewFile(metdata)
		} else {
			tm.ReicvCommand = false
		}

	}
}

func (g *Gossiper) PrintNextRound(nextRound uint32, confirmations []Confirmation) {
	conf := ""
	//origin1 <origin> ID1 <ID>, origin2 <origin> ID2 <ID>
	for i, c := range confirmations {
		if i < len(confirmations)-1{
			conf += fmt.Sprintf(" origin%d %s ID%d %d,", i+1, c.Origin, i+1, c.ID)

		}else{
			conf += fmt.Sprintf(" origin%d %s ID%d %d", i+1, c.Origin, i+1, c.ID)

		}
	}

	s := fmt.Sprintf("ADVANCING TO %d round BASED ON CONFIRMED MESSAGES%s\n", nextRound, conf)

	g.State.Mutex.Lock()
	g.State.RumorQueue = append(g.State.RumorQueue, packet.GossipPacket{Rumor:&packet.RumorMessage{
		Origin: g.Name,
		ID:     0,
		Text:   s,
	}})
	g.State.Mutex.Unlock()

	fmt.Printf(s)

}
