package gossip

import (
	"Peerster/helper"
	"Peerster/packet"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
)

type Gossiper struct {
	gossipAddr  string
	name        string
	peers       []*net.UDPAddr
	simple      bool
	connClient  *net.UDPConn
	connGossip  *net.UDPConn
	vectorClock map[string]uint32
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

	vectorClock := make(map[string]uint32)
	vectorClock[name] = 0

	return &Gossiper{
		gossipAddr:  gossipAddr,
		name:        name,
		peers:       peers,
		simple:      simple,
		connClient:  udpConnClient,
		connGossip:  udpConnGossip,
		vectorClock: vectorClock,
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
				//g.peers[peerAddr.String()] = peerAddr
				if !g.isPeer(peerAddr){
					g.peers = append(g.peers, peerAddr)
				}
				g.handleGossipPacket(receivedPacket, peerAddr)
			}
		}
	}
}
func (g *Gossiper) ListPeers(){
	str := ""
	for _, peer := range g.peers{
		str += peer.String()+","
	}
	fmt.Printf("PEERS %s\n", str[:len(str)-1])
}

func (g *Gossiper) isPeer(peerAddr *net.UDPAddr) bool{
	for _,addr := range g.peers{
		if addr.String() == peerAddr.String(){
			return true
		}
	}
	return false
}


