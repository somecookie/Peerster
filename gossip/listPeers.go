package gossip

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
)

type ListPeers struct{
	List  []*net.UDPAddr
	Mutex sync.Mutex
}

//selectPeerAtRandom selects a peer from the Peers map.
//It returns the key and the value
func (g *Gossiper) selectPeerAtRandom() *net.UDPAddr {
	g.Peers.Mutex.Lock()
	defer g.Peers.Mutex.Unlock()
	if len(g.Peers.List) == 0{
		return nil
	}
	return g.Peers.List[rand.Intn(len(g.Peers.List))]
}

func (g *Gossiper)AddPeer(peerAddr *net.UDPAddr){
	g.Peers.Mutex.Lock()
	defer g.Peers.Mutex.Unlock()
	if !g.isPeer(peerAddr) {
		g.Peers.List = append(g.Peers.List, peerAddr)
	}
}

//ListPeers output the message that lists all the Peers
func (g *Gossiper) ListPeers() {
	g.Peers.Mutex.Lock()
	defer g.Peers.Mutex.Unlock()
	str := ""
	for _, peer := range g.Peers.List {
		str += peer.String() + ","
	}
	fmt.Printf("PEERS %s\n", str[:len(str)-1])
}

//isPeer checks if a given address is already in the List of Peers
//This function does not handle concurrent access.
func (g *Gossiper) isPeer(peerAddr *net.UDPAddr) bool {
	if peerAddr.String() == g.gossipAddr {
		return false
	}
	for _, addr := range g.Peers.List {
		if addr.String() == peerAddr.String() {
			return true
		}
	}
	return false
}