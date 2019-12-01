package gossip

import (
	"github.com/somecookie/Peerster/fileSharing"
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
	GossipAddr      string
	Name            string
	Peers           PeersSet
	simple          bool
	connClient      *net.UDPConn
	connGossip      *net.UDPConn
	State           *GossiperState
	pendingACK      PendingACK
	counter         uint32
	antiEntropy     time.Duration
	rtimer          time.Duration
	DSDV            *routing.DSDV
	FilesIndex      *fileSharing.FilesIndex
	Requested       *fileSharing.DownloadState
	DSR             *packet.DuplicateSearchRequest
	fullMatches     *FullMatchCounter
	Matches         *Matches
	hoplimit        int
	TLCMajority     *TLCMajority
	stubbornTimeout int
}

//GossiperFactory creates a Gossiper from the parsed flags of main.go.
func GossiperFactory(gossipAddr, uiPort, name string, peers []*net.UDPAddr, simple bool, antiEntropy, rtimer, hoplimit, N,stubbornTimeout int) (*Gossiper, error) {

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

	pending := PendingACK{
		ACKS:  make(map[string]map[ACK]chan *packet.StatusPacket),
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
		State:       GossiperStateFactory(),
		pendingACK:  pending,
		counter:     0,
		antiEntropy: time.Duration(antiEntropy),
		rtimer:      time.Duration(rtimer),
		DSDV:        routing.DSDVFactory(),
		FilesIndex:  fileSharing.FilesIndexFactory(),
		Requested:   fileSharing.DownloadStateFactory(),
		DSR:         packet.DSRFactory(),
		fullMatches: &FullMatchCounter{
			Mutex:       sync.Mutex{},
			n: 0,
		},
		Matches:         MatchesFactory(),
		hoplimit:        hoplimit,
		TLCMajority:     TLCMajorityFactory(N),
		stubbornTimeout: stubbornTimeout,
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

	for _, peerStat := range g.State.VectorClock {
		if peerStat.Identifier == origin {
			return peerStat.NextID
		}
	}
	return 1
}

//Rumormongering forwards the given RumorMessage to a randomly selected peer
//In the case where flippedCoin is false, pastAddr should be nil.
//message is the rumor message we want to forward
//flipped coin tells if the rumor mongering was triggered by a coin flip
//pasAddr is the address of the node that sent us the message
//dstAddr is used when we want to send the rumor message to a given address
func (g *Gossiper) Rumormongering(message *packet.GossipPacket, flippedCoin bool, pastAddr *net.UDPAddr, dstAddr *net.UDPAddr) {

	g.Peers.Mutex.RLock()
	nbrPeers := len(g.Peers.Set)

	if nbrPeers == 0 || (nbrPeers == 1 && pastAddr != nil) {
		g.Peers.Mutex.RUnlock()
		return
	}
	peerAddr := g.SelectNewPeer(dstAddr, pastAddr)
	g.Peers.Mutex.RUnlock()

	g.sendMessage(message, peerAddr)

	if flippedCoin {
		packet.PrintFlippedCoin(peerAddr)
	} else {
		packet.PrintMongering(peerAddr)
	}
	go g.WaitForAck(message, peerAddr)
}

func (g *Gossiper) SelectNewPeer(dstAddr *net.UDPAddr, pastAddr *net.UDPAddr) *net.UDPAddr {
	peerAddr := dstAddr
	if dstAddr == nil {
		peerAddr = g.Peers.Random()

		if peerAddr == pastAddr {
			for peerAddr == pastAddr {
				peerAddr = g.Peers.Random()
			}
		}
	}
	return peerAddr
}

//AntiEntropyRoutine sends the anti-entropy Status Packet every g.antiEntropy seconds
func (g *Gossiper) AntiEntropyRoutine() {
	ticker := time.NewTicker(g.antiEntropy * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			peerAddr := g.Peers.Random()

			if peerAddr != nil {
				g.State.Mutex.RLock()
				g.sendStatusPacket(peerAddr)
				g.State.Mutex.RUnlock()
			}

		}
	}
}

//RouteRumorRoutine sends the route rumor message every g.rtimer seconds.
//It starts by sending a route rumor message.
func (g *Gossiper) RouteRumorRoutine() {

	peers := g.Peers.PeersSetAsList()

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
func (g *Gossiper) createNewRouteRumor() *packet.GossipPacket{
	atomic.AddUint32(&g.counter, 1)
	routeRumorMessage := &packet.RumorMessage{
		Origin: g.Name,
		ID:     g.counter,
		Text:   "",
	}

	gp := &packet.GossipPacket{Rumor:routeRumorMessage}

	g.State.UpdateGossiperState(gp)
	return gp
}


