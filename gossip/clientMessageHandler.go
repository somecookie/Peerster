package gossip

import (
	"encoding/hex"
	"github.com/somecookie/Peerster/packet"
	"strings"
	"sync/atomic"
	"time"
)

const MAX_BUDGET = 32

//HandleMessage is used to handle the messages that come from the client
func (g *Gossiper) HandleMessage(message *packet.Message) {

	if message.Destination == nil && message.File == nil && message.Request == nil && message.Text != "" {
		packet.PrintClientMessage(message)
		if g.simple {
			go g.sendSimpleMessage(message)
		} else {
			go g.startRumor(message)
		}

	} else if message.Destination != nil && message.File == nil && message.Request == nil && message.Text != "" {
		packet.PrintClientMessage(message)
		go g.startPrivate(message)
	} else if message.Destination == nil && message.File != nil && message.Request == nil && message.Text == "" {
		packet.PrintClientMessage(message)
		go g.IndexFile(*message.File)
	} else if message.File != nil && message.Request != nil && message.Text == "" {
		//packet.PrintClientMessage(message)

		if message.Destination == nil{
			if owners := g.Matches.GetOwners(*message.File, hex.EncodeToString(*message.Request),1); owners != nil{
				message.Destination = &owners[0]
			}else{
				return
			}
		}

		go g.startDownload(message)
	} else if message.Destination == nil && message.File == nil && message.Request == nil && message.Text == "" && message.Keywords != nil {
		go g.startSearchRequest(message)
	}
}

//startSearchRequest starts a new Search
func (g *Gossiper) startSearchRequest(message *packet.Message){
	keywords := strings.Split(*message.Keywords, ",")

	g.fullMatches.Lock()
	g.fullMatches.n = 0
	g.fullMatches.Unlock()

	g.Matches.Clear()

	if message.Budget != nil {
		sr := &packet.SearchRequest{
			Origin:   g.Name,
			Budget:   *message.Budget,
			Keywords: keywords,
		}
		g.SearchRequestRoutine(sr, nil)
	} else {
		budget := 2
		sr := &packet.SearchRequest{
			Origin:   g.Name,
			Budget:   uint64(budget),
			Keywords: keywords,
		}

		go g.SearchRequestRoutine(sr, nil)

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:

				g.fullMatches.Lock()
				if budget >= MAX_BUDGET || g.fullMatches.n== THRESHOLD_MATCHES {
					g.fullMatches.Unlock()
					return
				}
				g.fullMatches.Unlock()

				budget *= 2
				sr.Budget = uint64(budget)


				go g.SearchRequestRoutine(sr, nil)
			}
		}

	}
}

//startRumor starts a new rumor when the gossiper receives a message from the client which is not a private message.
func (g *Gossiper) startRumor(message *packet.Message) {

	atomic.AddUint32(&g.counter, 1)
	rumorMessage := &packet.RumorMessage{
		Origin: g.Name,
		ID:     g.counter,
		Text:   message.Text,
	}
	gp := &packet.GossipPacket{Rumor:rumorMessage}
	g.State.UpdateGossiperState(gp)

	g.Peers.Mutex.RLock()
	length := len(g.Peers.Set)
	g.Peers.Mutex.RUnlock()

	if length > 0 {
		g.Rumormongering(gp, false, nil, nil)
	}
}

//sendSimpleMessage handles the client's message when the simple flag is up
//It transform the Message in SimpleMessage and sends it to all its Peers
func (g *Gossiper) sendSimpleMessage(message *packet.Message) {
	simpleMessage := &packet.SimpleMessage{
		OriginalName:  g.Name,
		RelayPeerAddr: g.GossipAddr,
		Contents:      message.Text,
	}

	g.Peers.Mutex.RLock()
	defer g.Peers.Mutex.RUnlock()

	for _, addr := range g.Peers.Set {
		if addr == nil {
			continue
		}
		g.sendMessage(&packet.GossipPacket{Simple: simpleMessage}, addr)
	}
}

//startPrivate starts a private chat between g.Name and message.Destination
func (g *Gossiper) startPrivate(message *packet.Message) {

	pm := &packet.PrivateMessage{
		Origin:      g.Name,
		ID:          0,
		Text:        message.Text,
		Destination: *message.Destination,
		HopLimit:    packet.MaxHops - 1,
	}

	g.DSDV.Mutex.RLock()
	if g.DSDV.Contains(*message.Destination) {
		g.sendMessage(&packet.GossipPacket{Private: pm}, g.DSDV.NextHop[*message.Destination])
		g.DSDV.Mutex.RUnlock()

		g.State.Mutex.Lock()
		g.State.UpdatePrivateQueue(*message.Destination, pm)
		g.State.Mutex.Unlock()
	} else {
		g.DSDV.Mutex.RUnlock()
	}

}
