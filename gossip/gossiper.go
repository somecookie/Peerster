package gossip

import (
	"fmt"
	"github.com/somecookie/Peerster/helper"
	"github.com/somecookie/Peerster/packet"
	"net"
	"strings"
	"sync"
	"time"
)

type Gossiper struct {
	gossipAddr  string
	Name        string
	Peers       ListPeers
	simple      bool
	connClient  *net.UDPConn
	connGossip  *net.UDPConn
	RumorState  packet.RumorState
	pendingACK  PendingACK
	counter     uint32
	antiEntropy time.Duration
}

func (g *Gossiper) String() string {
	s := ""
	s += "Gossip Address: " + g.gossipAddr + "\n"
	s += "Name: " + g.Name + "\n"
	s += "Peers:\n"
	g.Peers.Mutex.Lock()
	for _, p := range g.Peers.List {
		s += "- " + p.String() + "\n"
	}
	g.Peers.Mutex.Lock()
	s += fmt.Sprintf("Counter: %d\n", g.counter)
	s += g.RumorState.String() + "\n"
	s += g.pendingACK.String() + "\n"
	return s
}

//BasicGossiperFactory creates a Gossiper from the parsed flags of main.go.
func BasicGossiperFactory(gossipAddr, uiPort, name string, peers []*net.UDPAddr, simple bool, antiEntropy int) (*Gossiper, error) {

	ipPort := strings.Split(gossipAddr, ":")
	if len(ipPort) != 2 {
		helper.HandleCrashingErr(&helper.IllegalArgumentError{
			ErrorMessage: "gossipAddress has the wrong format",
			Where:        "main.go",
		})
	}
	ip := ipPort[0]

	udpAddrClient, err := net.ResolveUDPAddr("udp4", ip+":"+uiPort)
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
		Mutex:            &sync.Mutex{},
	}

	pending := PendingACK{
		ACKS:  make(map[string][]ACK),
		mutex: sync.Mutex{},
	}

	listPeers := ListPeers{
		List:  peers,
		Mutex: sync.Mutex{},
	}

	return &Gossiper{
		gossipAddr:  gossipAddr,
		Name:        name,
		Peers:       listPeers,
		simple:      simple,
		connClient:  udpConnClient,
		connGossip:  udpConnGossip,
		RumorState:  rumorState,
		pendingACK:  pending,
		counter:     0,
		antiEntropy: time.Duration(antiEntropy),
	}, nil
}

//sendMessage sends the GossipPacket created by the gossiper based on the message received from the client
func (g *Gossiper) sendMessage(gossipPacket *packet.GossipPacket, addr *net.UDPAddr) {
	packetBytes, err := packet.GetPacketBytes(gossipPacket)
	helper.LogError(err)
	if err == nil {
		_, err := g.connGossip.WriteToUDP(packetBytes, addr)
		helper.LogError(err)
	}

}

func (g *Gossiper) HandleUDPClient() {

	defer g.connClient.Close()

	buffer := make([]byte, 1024)
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

func (g *Gossiper) HandleUPDGossiper() {
	defer g.connGossip.Close()
	go g.AntiEntropyRoutine()

	buffer := make([]byte, 1024)
	for {
		n, peerAddr, err := g.connGossip.ReadFromUDP(buffer)
		if err == nil {
			receivedPacket, err := packet.GetGossipPacket(buffer, n)

			if err == nil {
				g.AddPeer(peerAddr)
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

func (g *Gossiper) UpdateRumorState(message *packet.RumorMessage) {

	g.updateVectorClock(message)
	g.updateArchive(message)
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
	g.RumorState.MessageList = append(g.RumorState.MessageList,message)
	_, ok := g.RumorState.ArchivedMessages[message.Origin]

	if ok {
		g.RumorState.ArchivedMessages[message.Origin][message.ID] = message
	} else {
		g.RumorState.ArchivedMessages[message.Origin] = make(map[uint32]*packet.RumorMessage)
		g.RumorState.ArchivedMessages[message.Origin][message.ID] = message
	}
}

//Rumormongering forwards the given RumorMessage to a randomly selected peer
//It updates the RumorStatus of g.
//In the case where flippedCoin is false, pastAddr should be nil.
func (g *Gossiper) Rumormongering(message *packet.RumorMessage, flippedCoin bool, pastAddr *net.UDPAddr, dstAddr *net.UDPAddr) {

	g.Peers.Mutex.Lock()
	if len(g.Peers.List) == 0 || (len(g.Peers.List) == 1 && pastAddr != nil) {
		return
	}
	g.Peers.Mutex.Unlock()
	peerAddr := g.SelectNewPeer(dstAddr, pastAddr)
	packet.OutputOutRumorMessage(peerAddr)
	g.sendMessage(&packet.GossipPacket{Rumor: message}, peerAddr)

	if flippedCoin {
		packet.OutputFlippedCoin(peerAddr)
	}
	go g.WaitForAck(message, peerAddr)
}

func (g *Gossiper) SelectNewPeer(dstAddr *net.UDPAddr, pastAddr *net.UDPAddr) *net.UDPAddr {
	peerAddr := dstAddr
	if dstAddr == nil {
		peerAddr = g.selectPeerAtRandom()

		if peerAddr == pastAddr {
			for peerAddr == pastAddr {
				peerAddr = g.selectPeerAtRandom()
			}
		}
	}
	return peerAddr
}

func (g *Gossiper) AntiEntropyRoutine() {
	ticker := time.NewTicker(g.antiEntropy * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:

			if peerAddr := g.selectPeerAtRandom(); peerAddr != nil {
				g.RumorState.Mutex.Lock()
				g.sendStatusPacket(peerAddr)
				g.RumorState.Mutex.Unlock()
			}

		}
	}
}