package fileSharing

import (
	"github.com/somecookie/Peerster/packet"
	"sync"
)

//DownloadState is a structure used to keep track of the downloads/uploads
//The State fields maps the origin to a file metahash. Then the metahash is mapped to the index of the next chunk.
//At 0-index is the metafile.
type DownloadState struct {
	State map[string]map[string]uint32
	ACKs  map[string]map[string]chan *packet.DataReply //destination -> hash of chunk -> chan to know if chunk has been acked
	Mutex sync.RWMutex
}

func DownloadStateFactory() *DownloadState {
	return &DownloadState{
		State: make(map[string]map[string]uint32),
		ACKs:  make(map[string]map[string]chan  *packet.DataReply),
		Mutex: sync.RWMutex{},
	}
}

func (ds *DownloadState)GetAndIncrement(from, metahash string) uint32{
	n := ds.State[from][metahash]
	ds.State[from][metahash] += 1
	return n
}
