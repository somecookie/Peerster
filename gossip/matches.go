package gossip

import (
	"encoding/hex"
	"github.com/somecookie/Peerster/packet"
	"sync"
)

const THRESHOLD_MATCHES = 2

//FullMatchCounter is counter protected by a Mutex to count the full matches of a search
type FullMatchCounter struct {
	sync.Mutex
	n uint32
}

//match represents a match
type match struct {
	FileName string
	MetaHash string
}

//matchInfo is an inner struct used to gather the information received by the other peers during a search.
//chunkCount uint64 is the total number of chunk for this file
//chunkMap   map[uint64]string is a mapping of the index of a chunk to the peers who owns it.
type matchInfo struct {
	chunkCount uint64
	chunkMap   map[uint64][]string
}

//Matches records all the matches of the current research and all past full matches
//All operations of Matches are thread-safe.
//matches map[match]matchInfo is the mapping of a match (filename, metahash) to the information received from the other peers.
type Matches struct {
	sync.RWMutex
	matches map[match]matchInfo
	Queue   []struct {
		FileName string
		Origin   string
		MetaHash string
	}
}

func MatchesFactory() *Matches {
	return &Matches{
		RWMutex: sync.RWMutex{},
		matches: make(map[match]matchInfo),
		Queue: make([]struct {
			FileName string
			Origin   string
			MetaHash string
		}, 0, 2),
	}
}

//Clear removes all matches that are not full matches
func (ms *Matches) Clear() {
	ms.Lock()
	defer ms.Unlock()

	ms.matches = make(map[match]matchInfo)
}

//AddNewResult adds a new result from a given origin to the matches
//It returns whether or not this is a new result
func (ms *Matches) AddNewResult(result *packet.SearchResult, origin string) bool {
	ms.Lock()
	defer ms.Unlock()

	m := match{
		FileName: result.FileName,
		MetaHash: hex.EncodeToString(result.MetafileHash),
	}

	if _, ok := ms.matches[m]; !ok {
		ms.matches[m] = matchInfo{
			chunkCount: result.ChunkCount,
			chunkMap:   make(map[uint64][]string),
		}
	}

	return ms.matches[m].addInfo(result, origin)
}

//addInfo is an helper function that adds the result to the matchInfo
//It returns whether or not this is a new result
func (mi matchInfo) addInfo(result *packet.SearchResult, origin string) bool {

	newResult := false

	for _, index := range result.ChunkMap {
		if owners, ok := mi.chunkMap[index]; !ok {
			mi.chunkMap[index] = make([]string, 0)
			mi.chunkMap[index] = append(mi.chunkMap[index], origin)
			newResult = true
		} else if !isOwner(origin, owners) {
			newResult = true
			mi.chunkMap[index] = append(mi.chunkMap[index], origin)
		}

	}
	return newResult
}

func isOwner(owner string, owners []string) bool {
	for _, ow := range owners {
		if ow == owner {
			return true
		}
	}

	return false
}

//GetOwner retrieves the owner of the chunk indexed by index with the given file name  and metahash
func (ms *Matches) GetOwners(fileName, metahash string, index uint64) []string {
	ms.RLock()
	defer ms.RUnlock()
	m := match{
		FileName: fileName,
		MetaHash: metahash,
	}
	if _, ok := ms.matches[m]; ok {
		return ms.matches[m].chunkMap[index]
	} else {
		return nil
	}
}
