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

//NRandom chooses n random peers from the list of peers
//n uint64 is the number of peers selected at random
//exceptions ... *net.UDPAddr contains the peers that should not be chosen
//returns n randomly drawn peers, if the total number of peers is bigger than n, it returns all peers
func (ps *PeersSet)NRandom(n uint64, exceptions ... *net.UDPAddr) []*net.UDPAddr{

	subset := make([]*net.UDPAddr, 0, len(ps.Set))
	for _,v := range ps.Set{
		subset = append(subset, v)
	}

	if n > uint64(len(ps.Set)){
		return subset
	}

	perm := rand.Perm(len(ps.Set))
	result := make([]*net.UDPAddr,0,n)

	j := uint64(1)
	for _, i := range perm{
		if !isExcepted(subset[i], exceptions){
			result = append(result, subset[i])
			j++
		}

		if j == n{
			break
		}

	}

	return result
}
//isExcepted is an helper function used to know if addr is in the list of exception
//addr *net.UDPAddr is the address of the peer
//exceptions []*net.UDPAddr is the list of exceptions
func isExcepted(addr *net.UDPAddr, exceptions []*net.UDPAddr) bool{
	for _, ex := range exceptions{
		if addr.String() == ex.String(){
			return true
		}
	}

	return false
}


//PeersSetAsList returns the values of the PeersSet as a list of *net.UDPAddr
func (ps* PeersSet) PeersSetAsList() []*net.UDPAddr{
	ls := make([]*net.UDPAddr, 0, len(ps.Set))
	for _,addr := range ps.Set{
		ls = append(ls, addr)
	}
	return ls
}

//PeersSetAsList returns the values of the PeersSet as a list of ip:port (as string)
func (ps PeersSet) PeersAsStringList() []string {
	keys := make([]string, 0, len(ps.Set))
	for k := range ps.Set {
		keys = append(keys, k)
	}
	return keys
}

func PrintPeers(g *Gossiper) {
	//fmt.Printf("PEERS %s\n", g.Peers.String())
}

