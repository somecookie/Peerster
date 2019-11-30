package packet

import (
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
)

//SearchRequest are the messages search for a file
//Origin   string is the node that starts the request
//Budget   uint64 is the amount of work still available for this request
//Keywords []string is the list of searched keywords
type SearchRequest struct {
	Origin   string
	Budget   uint64
	Keywords []string
}

//SearchReply are the replies sent when one of the keywords of a request match an indexed file
//Origin      string is the name of the node replying
//Destination string is the name of the node who searched for some keywords
//HopLimit    uint32 is the TTL of this packet
//Results     []*SearchResult are the results of the search
type SearchReply struct {
	Origin      string
	Destination string
	HopLimit    uint32
	Results     []*SearchResult
}

//SearchResult is the structure used to send the result of a search when there is a match
//FileName     string is the name of the file
//MetafileHash []byte is the metahash
//ChunkMap     []uint64 is a list of the index of the chunks the peers has locally
//ChunkCount   uint64 is the number of chunks of the file
type SearchResult struct {
	FileName     string
	MetafileHash []byte
	ChunkMap     []uint64
	ChunkCount   uint64
}

//PrintSearchReply print the required message(s) when a reply is received
func PrintSearchResult(result *SearchResult, origin string) {
	chunkList := ""
	for i := 0; i < len(result.ChunkMap); i++ {
		if i != len(result.ChunkMap)-1 {
			chunkList += fmt.Sprintf("%d,", result.ChunkMap[i])
		} else {
			chunkList += fmt.Sprintf("%d", result.ChunkMap[i])
		}
	}

	fmt.Printf("FOUND match %s at %s metafile=%s chunks=%s\n", result.FileName, origin, hex.EncodeToString(result.MetafileHash), chunkList)
}

//DuplicateSearchRequest is a concurrent set (i.e., thread safe) structures that keeps
//track of the already received search requests.
//Received map[Duplicates]bool is the internal representation of the set
type DuplicateSearchRequest struct {
	received map[duplicate]bool
	sync.RWMutex
}

//duplicate is an internal structure used as a key for the DuplicateSearchRequest map
type duplicate struct {
	Origin   string
	Keywords string
}

//DSRFactory creates a new DuplicateSearchRequest struct
func DSRFactory() *DuplicateSearchRequest {
	return &DuplicateSearchRequest{
		received: make(map[duplicate]bool),
		RWMutex:  sync.RWMutex{},
	}
}

//Add adds a search request to the set of duplicates
func (dsr *DuplicateSearchRequest) Add(sr *SearchRequest) {
	dsr.Lock()
	defer dsr.Unlock()

	dup := duplicate{
		Origin:   sr.Origin,
		Keywords: strings.Join(sr.Keywords, ","),
	}

	if _, ok := dsr.received[dup]; !ok {
		dsr.received[dup] = true
	}
}

func (dsr *DuplicateSearchRequest) Remove(sr *SearchRequest) {
	dsr.Lock()
	defer dsr.Unlock()

	dup := duplicate{
		Origin:   sr.Origin,
		Keywords: strings.Join(sr.Keywords, ","),
	}

	if _, ok := dsr.received[dup]; ok {
		delete(dsr.received, dup)
	}
}

func (dsr *DuplicateSearchRequest) Contains(sr *SearchRequest) bool {
	dsr.RLock()
	defer dsr.RUnlock()

	dup := duplicate{
		Origin:   sr.Origin,
		Keywords: strings.Join(sr.Keywords, ","),
	}

	_, ok := dsr.received[dup]

	return ok
}
