package gossip

import (
	"Peerster/helper"
	"Peerster/packet"
	"fmt"
	"github.com/dedis/protobuf"
	"net"
	"strings"
	"sync"
)

type Gossiper struct {
	gossipAddr string
	name       string
	peers      map[string]*net.UDPAddr
	simple     bool
	connClient *net.UDPConn
	connGossip *net.UDPConn
}

//BasicGossiperFactory creates a Gossiper from the parsed flags of main.go.
func BasicGossiperFactory(gossipAddr, uiPort, name string, peers map[string]*net.UDPAddr, simple bool) (*Gossiper, error) {

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

	return &Gossiper{
		gossipAddr: gossipAddr,
		name:       name,
		peers:      peers,
		simple:     simple,
		connClient: udpConnClient,
		connGossip: udpConnGossip,
	}, nil
}

func (g *Gossiper) HandleUDPClient(group *sync.WaitGroup) {
	defer group.Done()
	defer g.connClient.Close()

	buffer := make([]byte, 1024)
	for {
		n, _, err := g.connClient.ReadFromUDP(buffer)
		helper.LogError(err)

		if err == nil {
			receivedPacket := &packet.GossipPacket{}
			err := protobuf.Decode(buffer[:n], receivedPacket)
			helper.LogError(err)
			if err == nil {
				g.handleClientSimpleMessage(receivedPacket)
			}
		}
	}

}

func (g *Gossiper) handleClientSimpleMessage(receivedPacket *packet.GossipPacket) {
	message := receivedPacket.Simple
	fmt.Printf("CLIENT MESSAGE %s\n", message.Contents)
	message.OriginalName = g.name
	message.RelayPeerAddr = g.gossipAddr

	for _, addr := range g.peers {
		if addr == nil {
			continue
		}

		packetBytes, err := packet.GetPacketBytes(&packet.GossipPacket{Simple:message})
		helper.LogError(err)

		if err == nil {

			helper.LogError(err)

			if err == nil {
				_, err := g.connGossip.WriteToUDP(packetBytes, addr)
				helper.LogError(err)
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
			receivedPacket := &packet.GossipPacket{}
			err := protobuf.Decode(buffer[:n], receivedPacket)
			helper.LogError(err)

			if err == nil {
				g.peers[receivedPacket.Simple.RelayPeerAddr] = peerAddr
				g.handleGossiperSimpleMessage(receivedPacket)
			}
		}
	}
}

func (g *Gossiper) handleGossiperSimpleMessage(receivedPacket *packet.GossipPacket) {
	message := receivedPacket.Simple
	//SIMPLE MESSAGE origin E from 127.0.0.1:$relayPort contents $message
	fmt.Printf("SIMPLE MESSAGE origin %s from %s contents %s\n", message.OriginalName, message.RelayPeerAddr, message.Contents)
	peerAddr := g.peers[message.RelayPeerAddr]
	var peersStr string = ""
	message.RelayPeerAddr = g.gossipAddr

	for s, addr := range g.peers {
		peersStr += s+","
		if addr == nil || addr == peerAddr {
			continue
		}
		packetBytes, err := packet.GetPacketBytes(&packet.GossipPacket{Simple:message})

		if err == nil {
			_, err := g.connGossip.WriteToUDP(packetBytes, addr)
			helper.LogError(err)
		}
	}
	fmt.Printf("PEERS %s\n", peersStr[:len(peersStr)-1])
}
