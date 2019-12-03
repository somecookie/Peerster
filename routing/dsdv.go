package routing

import (
	"fmt"
	"net"
	"sync"
)

//This structures contains all the needed data structures to implement the DSDV routing scheme.
//The methods of DSDV are not thread-safe.
//NextHop is the next-hop routing table. It maps the destination to the next hop.
//DstSeq maps a destination to it destination sequence number.
//The destination sequence number is the highest ID of the RumorMessage coming from the destination.
type DSDV struct{
	NextHop map[string]*net.UDPAddr
	DstSeq map[string]uint32
	Mutex sync.RWMutex
}

//DSDVFactory is a factory to create a new empty DSDV.
func DSDVFactory() *DSDV {
	return &DSDV{
		NextHop: make(map[string]*net.UDPAddr),
		DstSeq:  make(map[string]uint32),
		Mutex:   sync.RWMutex{},
	}
}

//Contains verifies if the given origin is already in the next-hop routing table
func (dsdv *DSDV) Contains(origin string) bool{
	_, ok := dsdv.NextHop[origin]
	return ok
}

//Update updates the next-hop table and the destination sequence number.
//id is the ID of the newly arrived rumorMessage or TLCMessage.
//origin is the origin of the newly arrived rumorMessage or TLCMessage.
//text is the content of the rumorMessage
//from is the address from which the rumor message arrived.
func (dsdv *DSDV) Update(id uint32, origin, text string, from *net.UDPAddr){
	_, ok := dsdv.NextHop[origin]

	if !ok{
		dsdv.NextHop[origin] = from
		dsdv.DstSeq[origin] = id

		if text != ""{
			PrintUpdateDSVD(origin, from)
		}

	}else{
		highestSeq := dsdv.DstSeq[origin]
		if highestSeq < id {
			dsdv.DstSeq[origin] = id
			dsdv.NextHop[origin] = from

			if text != ""{
				PrintUpdateDSVD(origin, from)
			}
		}
	}
}

//GetOrigins retrieves a list of node Origins identifier.
func (dsdv *DSDV) GetOrigins() []string{
	origins := make([]string, 0, len(dsdv.NextHop))
	for origin, _ := range dsdv.NextHop{
		origins = append(origins, origin)
	}

	return origins
}

//PrintUpdateDSVD prints the message "DSDV <peer_name> <ip:port> when the the DSDV routing table is updated.
func PrintUpdateDSVD(origin string, from *net.UDPAddr){
	fmt.Printf("DSDV %s %s\n", origin, from.String())
}

//DSDV implements the function of the interface String
func (dsdv *DSDV) String() string{
	s := "Origin - Next-Hop - Sequence-Number\n"
	for origin, nexthop := range dsdv.NextHop{
		s += fmt.Sprintf("%s - %s - %d\n", origin, nexthop.String(), dsdv.DstSeq[origin])
	}

	return s[:len(s)-1]
}