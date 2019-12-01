package gossip

import (
	"fmt"
	"github.com/somecookie/Peerster/blockchain"
	"github.com/somecookie/Peerster/fileSharing"
	"github.com/somecookie/Peerster/packet"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

//TLCMajority represents the state for the majorities. All operations on this structure are thread safe.
//Acks         map[uint32][]string is the mapping between an ID and all the peers that have already acknowledged the TLCMessage with this ID
//AcksChannels map[uint32]chan bool is the mapping an ID of a TLCMessage and its associated channel
//Total        int is the total number of Peers in the network. Therefore, the majority is Total/2.
type TLCMajority struct {
	sync.RWMutex
	Acks         map[uint32][]string
	AcksChannels map[uint32]chan bool
	Total        int
}

func TLCMajorityFactory(N int) *TLCMajority {
	return &TLCMajority{
		RWMutex:      sync.RWMutex{},
		Acks:         make(map[uint32][]string),
		AcksChannels: make(map[uint32]chan bool),
		Total:        N,
	}
}

//SelfAdd allows the gossiper to add itself when it creates a new TLCMessage.
//ID uint32 is the ID of the newly created TLCMessage
//selfName string is the name of the gossiper
func (tm *TLCMajority) SelfAdd(ID uint32, selfName string) {
	tm.Lock()
	defer tm.Unlock()

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
	tm.Lock()
	defer tm.Unlock()

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
	tm.RLock()
	defer tm.RUnlock()
	peersList := tm.Acks[uint32(confirmed.Confirmed)]
	witnesses := strings.Join(peersList, ",")
	fmt.Printf("RE-BROADCAST ID %d WITNESSES %s\n", confirmed.Confirmed, witnesses)
}

//GetChannel allows you to get the channel corresponding to the given id. It returns nil if the ID is not valid.
func (tm *TLCMajority) GetChannel(ID uint32) chan bool {
	tm.RLock()
	defer tm.RUnlock()
	if c, ok := tm.AcksChannels[ID]; ok {
		return c
	} else {
		return nil
	}
}

//Stubborn stubbornly rebroadcast the tlcMessage after a timeout or, if a majority is reached before the timeout
//re-broadcasts the new confirmed message.
//tlcMessage *packet.TLCMessage is the new TLC message broadcast in the first place.
func (g *Gossiper) Stubborn(tlcMessage *packet.TLCMessage) {
	ticker := time.NewTicker(time.Duration(g.stubbornTimeout) * time.Second)

	ackChannel := g.TLCMajority.GetChannel(tlcMessage.ID)

	if ackChannel == nil {
		return
	}

	for {
		select {
		case <-ticker.C:
			log.Println("Mongering stubborn")
			g.Rumormongering(&packet.GossipPacket{TLCMessage: tlcMessage}, false, nil, nil)
		case <-ackChannel:
			ticker.Stop()
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
			g.TLCMajority.PrintReBroadcast(confirmation)
			gp := &packet.GossipPacket{TLCMessage: confirmation}
			g.State.UpdateGossiperState(gp)
			log.Println("Mongering Majority")
			g.Rumormongering( gp,false, nil, nil)
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
	atomic.AddUint32(&g.counter, 1)
	tlcMsg := &packet.TLCMessage{
		Origin:      g.Name,
		ID:          g.counter,
		Confirmed:   -1,
		TxBlock:     bp,
		VectorClock: nil,
		Fitness:     0,
	}
	gp := &packet.GossipPacket{TLCMessage: tlcMsg}
	g.TLCMajority.SelfAdd(tlcMsg.ID, g.Name)
	g.State.UpdateGossiperState(gp)

	log.Println("Mongering new file")
	g.Rumormongering(gp, false, nil, nil)
	go g.Stubborn(tlcMsg)
}

//This function handles the TLCMessage when they are received by the gossiper
//tlcMessage *packet.TLCMessage is the TLCMessage received by the gossiper
func (g *Gossiper) HandleTLCMessage(tlcMessage *packet.TLCMessage) {
	//check validity

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

}

//TLCAckRoutine handles the incoming acks, either by updating the majority counter if
//the gossiper is the destination or by forwarding the ack to the next hop
//ack *packet.TLCAck is the received ack
func (g *Gossiper) TLCAckRoutine(ack *packet.TLCAck) {
	if ack.Destination == g.Name {
		log.Printf("Ack from %s\n", ack.Origin)
		g.TLCMajority.AddNewAck(ack)
	} else if ack.HopLimit > 0 {

		log.Printf("%s is hop between %s and %s\n", g.Name, ack.Origin, ack.Destination)

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
