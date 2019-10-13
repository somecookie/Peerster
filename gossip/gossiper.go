package gossip

import (
	"fmt"
	"github.com/somecookie/Peerster/helper"
	"github.com/somecookie/Peerster/packet"
	"math/rand"
	"net"
	"strings"
	"sync"
)

type Gossiper struct {
	gossipAddr string
	name       string
	peers      []*net.UDPAddr
	simple     bool
	connClient *net.UDPConn
	connGossip *net.UDPConn
	rumorState packet.RumorState
	pendingACK PendingACK
	counter    uint32
}

func (g *Gossiper) String() string {
	s := ""
	s += "Gossip Address: " + g.gossipAddr + "\n"
	s += "Name: " + g.name + "\n"
	s += "Peers:\n"
	for _, p := range g.peers {
		s += "- " + p.String() + "\n"
	}
	s += fmt.Sprintf("Counter: %d\n", g.counter)
	s += g.rumorState.String() + "\n"
	s += g.pendingACK.String() + "\n"
	return s
}

//BasicGossiperFactory creates a Gossiper from the parsed flags of main.go.
func BasicGossiperFactory(gossipAddr, uiPort, name string, peers []*net.UDPAddr, simple bool) (*Gossiper, error) {

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
		Mutex:            &sync.Mutex{},
	}

	pending := PendingACK{
		ACKS:  make(map[string][]ACK),
		mutex: sync.Mutex{},
	}

	return &Gossiper{
		gossipAddr: gossipAddr,
		name:       name,
		peers:      peers,
		simple:     simple,
		connClient: udpConnClient,
		connGossip: udpConnGossip,
		rumorState: rumorState,
		pendingACK: pending,
		counter:    0,
	}, nil
}

//selectPeerAtRandom selects a peer from the peers map.
//It returns the key and the value
func (g *Gossiper) selectPeerAtRandom() *net.UDPAddr {
	return g.peers[rand.Intn(len(g.peers))]
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

func (g *Gossiper) HandleUDPClient(group *sync.WaitGroup) {
	defer group.Done()
	defer g.connClient.Close()

	buffer := make([]byte, 1024)
	for {
		n, _, err := g.connClient.ReadFromUDP(buffer)
		helper.LogError(err)

		if err == nil {
			message, err := packet.GetMessage(buffer, n)
			if err == nil {
				g.handleMessage(message)
			}
		}
	}
}

func (g *Gossiper) HandleUPDGossiper(group *sync.WaitGroup) {
	defer group.Done()
	defer g.connGossip.Close()

	buffer := make([]byte, 1024)
	for {
		n, peerAddr, err := g.connGossip.ReadFromUDP(buffer)
		if err == nil {
			receivedPacket, err := packet.GetGossipPacket(buffer, n)

			if err == nil {
				if !g.isPeer(peerAddr) {
					g.peers = append(g.peers, peerAddr)
				}
				g.handleGossipPacket(receivedPacket, peerAddr)
			}
		}
	}
}

//ListPeers output the message that lists all the peers
func (g *Gossiper) ListPeers() {
	str := ""
	for _, peer := range g.peers {
		str += peer.String() + ","
	}
	fmt.Printf("PEERS %s\n", str[:len(str)-1])
}

//isPeer checks if a given address is already in the list of peers
func (g *Gossiper) isPeer(peerAddr *net.UDPAddr) bool {
	if peerAddr.String() == g.gossipAddr {
		return false
	}

	for _, addr := range g.peers {
		if addr.String() == peerAddr.String() {
			return true
		}
	}
	return false
}

func (g *Gossiper) GetNextID(origin string) uint32 {

	for _, peerStat := range g.rumorState.VectorClock {
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
	for i, peerStat := range g.rumorState.VectorClock {
		if peerStat.Identifier == message.Origin {
			if message.ID == peerStat.NextID {
				g.rumorState.VectorClock[i].NextID += 1
			}
			return
		}
	}
	var nextID uint32
	if message.ID == 1 {
		nextID = 2
	} else {
		nextID = 1
	}
	g.rumorState.VectorClock = append(g.rumorState.VectorClock, packet.PeerStatus{
		Identifier: message.Origin,
		NextID:     nextID,
	})
}

func (g *Gossiper) updateArchive(message *packet.RumorMessage) {
	_, ok := g.rumorState.ArchivedMessages[message.Origin]

	if ok {
		g.rumorState.ArchivedMessages[message.Origin][message.ID] = message
	} else {
		g.rumorState.ArchivedMessages[message.Origin] = make(map[uint32]*packet.RumorMessage)
		g.rumorState.ArchivedMessages[message.Origin][message.ID] = message
	}
}

//Rumormongering forwards the given RumorMessage to a randomly selected peer
//It updates the RumorStatus of g.
func (g *Gossiper) Rumormongering(message *packet.RumorMessage, flippedCoin bool) {

	peerAddr := g.selectPeerAtRandom()
	g.sendMessage(&packet.GossipPacket{Rumor: message}, peerAddr)

	if flippedCoin {
		packet.OutputFlippedCoin(peerAddr)
	}
	packet.OutputOutRumorMessage(peerAddr)
	go g.WaitForAck(message, peerAddr)
}

