package gossip

import (
	"github.com/somecookie/Peerster/helper"
	"github.com/somecookie/Peerster/packet"
	"github.com/somecookie/Peerster/routing"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Gossiper struct {
	GossipAddr  string
	Name        string
	Peers       PeersSet
	simple      bool
	connClient  *net.UDPConn
	connGossip  *net.UDPConn
	RumorState  packet.RumorState
	pendingACK  PendingACK
	counter     uint32
	antiEntropy time.Duration
	rtimer      time.Duration
	DSDV        routing.DSDV
}

//GossiperFactory creates a Gossiper from the parsed flags of main.go.
func GossiperFactory(gossipAddr, uiPort, name string, peers []*net.UDPAddr, simple bool, antiEntropy int, rtimer int) (*Gossiper, error) {

	ipPort := strings.Split(gossipAddr, ":")
	if len(ipPort) != 2 {
		helper.HandleCrashingErr(&helper.IllegalArgumentError{
			ErrorMessage: "gossipAddress has the wrong format",
			Where:        "main.go",
		})
	}

	udpAddrClient, err := net.ResolveUDPAddr("udp4", "127.0.0.1:"+uiPort)
	if err != nil {
		return nil, err
	}

	udpAddrGossip, err := net.ResolveUDPAddr("udp4", gossipAddr)
	if err != nil {
		return nil, err
	}

	udpConnGossip, err := net.ListenUDP("udp4", udpAddrGossip)
	if err != nil {
		return nil, err
	}

	udpConnClient, err := net.ListenUDP("udp4", udpAddrClient)
	if err != nil {
		return nil, err
	}

	vectorClock := make([]packet.PeerStatus, 0)
	archivedMessage := make(map[string]map[uint32]*packet.RumorMessage)
	rumorState := packet.RumorState{
		VectorClock:      vectorClock,
		ArchivedMessages: archivedMessage,
		MessageList:      make([]*packet.RumorMessage, 0),
		Mutex:            sync.RWMutex{},
	}
	pending := PendingACK{
		ACKS:  make(map[string]map[ACK]bool),
		Mutex: sync.RWMutex{},
	}

	peersSet := PeersSet{
		Set:   make(map[string]*net.UDPAddr),
		Mutex: sync.RWMutex{},
	}

	for _, addr := range peers {
		if addr.String() != gossipAddr {
			peersSet.Add(addr)
		}
	}

	return &Gossiper{
		GossipAddr:  gossipAddr,
		Name:        name,
		Peers:       peersSet,
		simple:      simple,
		connClient:  udpConnClient,
		connGossip:  udpConnGossip,
		RumorState:  rumorState,
		pendingACK:  pending,
		counter:     0,
		antiEntropy: time.Duration(antiEntropy),
		rtimer:      time.Duration(rtimer),
		DSDV:        routing.DSDVFactory(),
	}, nil
}

//sendMessage sends the GossipPacket created by the gossiper based on the message received from the client
//GossipPacket is the packet we want to send
//dest is the destination address
func (g *Gossiper) sendMessage(gossipPacket *packet.GossipPacket, dest *net.UDPAddr) {
	packetBytes, err := packet.GetPacketBytes(gossipPacket)
	helper.LogError(err)
	if err == nil {
		_, err := g.connGossip.WriteToUDP(packetBytes, dest)
		helper.LogError(err)
	}

}

func (g *Gossiper) ClientListener() {

	defer g.connClient.Close()

	buffer := make([]byte, 10000)
	for {
		n, _, err := g.connClient.ReadFromUDP(buffer)
		helper.LogError(err)

		if err == nil {
			message, err := packet.GetMessage(buffer, n)
			if err == nil {
				g.HandleMessage(message)
			}
		}
	}
}

func (g *Gossiper) GossiperListener() {
	defer g.connGossip.Close()

	buffer := make([]byte, 10000)
	for {
		n, peerAddr, err := g.connGossip.ReadFromUDP(buffer)
		if err == nil {
			receivedPacket, err := packet.GetGossipPacket(buffer, n)

			if err == nil {
				if peerAddr.String() != g.GossipAddr {
					g.Peers.Mutex.Lock()
					g.Peers.Add(peerAddr)
					g.Peers.Mutex.Unlock()
				}

				g.GossipPacketHandler(receivedPacket, peerAddr)
			}
		}
	}
}

func (g *Gossiper) GetNextID(origin string) uint32 {

	for _, peerStat := range g.RumorState.VectorClock {
		if peerStat.Identifier == origin {
			return peerStat.NextID
		}
	}
	return 1
}

//UpdateRumorState updates the vector clock and the archives of g.
//This method is thread-safe, no need to lock when using it.
func (g *Gossiper) UpdateRumorState(message *packet.RumorMessage) {
	g.RumorState.Mutex.Lock()
	g.updateVectorClock(message)
	g.RumorState.Mutex.Unlock()

	g.RumorState.Mutex.Lock()
	g.updateArchive(message)
	g.RumorState.Mutex.Unlock()
}

func (g *Gossiper) updateVectorClock(message *packet.RumorMessage) {
	inVC := false
	for i, peerStat := range g.RumorState.VectorClock {
		if peerStat.Identifier == message.Origin {
			inVC = true
			if message.ID == peerStat.NextID {
				g.RumorState.VectorClock[i].NextID += 1
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
		g.RumorState.VectorClock = append(g.RumorState.VectorClock, packet.PeerStatus{
			Identifier: message.Origin,
			NextID:     nextID,
		})
	}

}

func (g *Gossiper) updateArchive(message *packet.RumorMessage) {
	if message.Text != "" {
		g.RumorState.MessageList = append(g.RumorState.MessageList, message)
	}

	_, ok := g.RumorState.ArchivedMessages[message.Origin]

	if ok {
		g.RumorState.ArchivedMessages[message.Origin][message.ID] = message
	} else {
		g.RumorState.ArchivedMessages[message.Origin] = make(map[uint32]*packet.RumorMessage)
		g.RumorState.ArchivedMessages[message.Origin][message.ID] = message
	}
}

//Rumormongering forwards the given RumorMessage to a randomly selected peer
//In the case where flippedCoin is false, pastAddr should be nil.
//message is the rumor message we want to forward
//flipped coin tells if the rumor mongering was triggered by a coin flip
//pasAddr is the address of the node that sent us the message
//dstAddr is used when we want to send the rumor message to a given address
func (g *Gossiper) Rumormongering(message *packet.RumorMessage, flippedCoin bool, pastAddr *net.UDPAddr, dstAddr *net.UDPAddr) {
	g.Peers.Mutex.RLock()
	nbrPeers := len(g.Peers.Set)
	g.Peers.Mutex.RUnlock()

	/*if nbrPeers == 0 || (nbrPeers == 1 && pastAddr != nil) {
			return
	}

	peerAddr := g.SelectNewPeer(dstAddr, pastAddr)
	*/

	if nbrPeers == 0 {
		return
	}

	peerAddr := dstAddr

	if dstAddr == nil {
		g.Peers.Mutex.RLock()
		peerAddr = g.Peers.Random()
		g.Peers.Mutex.RUnlock()
	}

	packet.PrintMongering(peerAddr)
	g.sendMessage(&packet.GossipPacket{Rumor: message}, peerAddr)

	if flippedCoin {
		packet.PrintFlippedCoin(peerAddr)
	}
	go g.WaitForAck(message, peerAddr)
}

/*//SelectNewPeer chooses a new peer that is different than the last chosen peer
//This method is thread-safe.
func (g *Gossiper) SelectNewPeer(dstAddr *net.UDPAddr, pastAddr *net.UDPAddr) *net.UDPAddr {
	peerAddr := dstAddr
	if dstAddr == nil {
		g.Peers.Mutex.RLock()
		peerAddr = g.Peers.Random()
		g.Peers.Mutex.RUnlock()

		if peerAddr == pastAddr {
			for peerAddr == pastAddr {
				g.Peers.Mutex.RLock()
				peerAddr = g.Peers.Random()
				g.Peers.Mutex.RUnlock()
			}
		}
	}
	return peerAddr
}*/

//AntiEntropyRoutine sends the anti-entropy Status Packet every g.antiEntropy seconds
func (g *Gossiper) AntiEntropyRoutine() {
	ticker := time.NewTicker(g.antiEntropy * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:

			g.Peers.Mutex.RLock()
			peerAddr := g.Peers.Random()
			g.Peers.Mutex.RUnlock()

			if peerAddr != nil {
				g.RumorState.Mutex.RLock()
				g.sendStatusPacket(peerAddr)
				g.RumorState.Mutex.RUnlock()
			}

		}
	}
}

//RouteRumorRoutine sends the route rumor message every g.rtimer seconds.
//It starts by sending a route rumor message.
func (g *Gossiper) RouteRumorRoutine() {

	g.Peers.Mutex.RLock()
	peers := g.Peers.PeersSetAsList()
	g.Peers.Mutex.RUnlock()

	routeRumorMessage := g.createNewRouteRumor()

	for _, peer := range peers {
		g.Rumormongering(routeRumorMessage, false, nil, peer)
	}

	ticker := time.NewTicker(g.rtimer * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			routeRumorMessage := g.createNewRouteRumor()
			g.Rumormongering(routeRumorMessage, false, nil, nil)
		}
	}

}

//createNewRouteRumor creates a new route rumor message
func (g *Gossiper) createNewRouteRumor() *packet.RumorMessage {
	atomic.AddUint32(&g.counter, 1)
	routeRumorMessage := &packet.RumorMessage{
		Origin: g.Name,
		ID:     g.counter,
		Text:   "",
	}
	g.UpdateRumorState(routeRumorMessage)
	return routeRumorMessage
}
