package gossip

import (
	"Peerster/helper"
	"Peerster/packet"
	"fmt"
)

func (g *Gossiper) handleGossiperSimpleMessage(receivedPacket *packet.GossipPacket) {
	message := receivedPacket.Simple
	fmt.Printf("SIMPLE MESSAGE origin %s from %s contents %s\n", message.OriginalName, message.RelayPeerAddr, message.Contents)
	peerAddr := g.peers[message.RelayPeerAddr]
	var peersStr = ""
	message.RelayPeerAddr = g.gossipAddr

	for s, addr := range g.peers {
		peersStr += s + ","
		if addr == nil || addr == peerAddr {
			continue
		}
		packetBytes, err := packet.GetPacketBytes(&packet.GossipPacket{Simple: message})

		if err == nil {
			_, err := g.connGossip.WriteToUDP(packetBytes, addr)
			helper.LogError(err)
		}
	}
	fmt.Printf("PEERS %s\n", peersStr[:len(peersStr)-1])
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

		packetBytes, err := packet.GetPacketBytes(&packet.GossipPacket{Simple: message})
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

