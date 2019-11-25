package fileSharing

import (
	"encoding/hex"
	"github.com/somecookie/Peerster/packet"
	"strings"
	"sync"
)

//FilesIndex is the index of the files shared by the gossiper.
//The field FilesIndex is a mapping from the metahash of a file to its metadata.
//Note that the operations on FilesIndex are not thread-safe.
type FilesIndex struct {
	Index map[string]*Metadata
	Mutex sync.RWMutex
}

func FilesIndexFactory() *FilesIndex{
	return &FilesIndex{
		Index: make(map[string]*Metadata),
		Mutex: sync.RWMutex{},
	}
}

//Store the metadata of the file to the index
func (fi *FilesIndex) Store(metadata *Metadata){
	fi.Index[hex.EncodeToString(metadata.MetaHash)] = metadata
}

//FindChunkFromHash takes a hash of a chunk and find the chunk in the index.
//If the hash is the metahash, it returns the metafile
//It returns nil if this chunk is not in the index
func (fi *FilesIndex)FindChunkFromHash(hash string) []byte{
	for metahash, metadata := range fi.Index{
		if hash == metahash{
			return metadata.Metafile
		}

		if chunk, ok := metadata.Chunks[hash]; ok{
			return chunk
		}
	}
	return nil
}

func (fi *FilesIndex) FindMatchingFiles(keywords []string) []*packet.SearchResult{
	results := make([]*packet.SearchResult,0)
	for _, keyword := range keywords{
		for _, metadata := range fi.Index{
			if strings.Contains(metadata.Name, keyword){
				results = append(results, &packet.SearchResult{
					FileName:     metadata.Name,
					MetafileHash: metadata.MetaHash,
					ChunkMap:     nil, //TODO
					ChunkCount:   0,
				})
			}
		}
	}
	return results
}
