package gossip

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
)

//PeerSet is a set of *net.UDPAddr that corresponds to the addresses of known peers of some gossiper.
//All methods on the PeerSet are not thread-safe. You should use the Mutex that is link to it.
type PeersSet struct {
	Set   map[string]*net.UDPAddr
	Mutex sync.RWMutex
}

//String returns a string of the addresses as comma separated ip:port
func (ps *PeersSet) String() string {

	if len(ps.Set) == 0{
		return ""
	}

	str := ""
	for peerAddr := range ps.Set {
		str += fmt.Sprintf("%s,", peerAddr)
	}
	return str[:len(str)-1]
}

//Contains checks if an address is in the set
func (ps *PeersSet) Contains(peerAddr *net.UDPAddr) bool {
	_, ok := ps.Set[peerAddr.String()]
	return ok
}

//Add adds new address to the set
func (ps *PeersSet) Add(peerAddr *net.UDPAddr) {
	ps.Set[peerAddr.String()] = peerAddr
}

//selectPeerAtRandom selects a peer from the Peers map.
//It returns the key and the value
func (ps *PeersSet) Random() *net.UDPAddr {

	if len(ps.Set) == 0 {
		return nil
	}

	r := rand.Intn(len(ps.Set))

	for _,addr := range ps.Set {
		if r == 0 {
			return addr
		} else {
			r -= 1
		}
	}
	return nil
}

func PrintPeers(g *Gossiper) {
	fmt.Printf("PEERS %s\n", g.Peers.String())
}

