package fileSharing

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/somecookie/Peerster/helper"
	"io"
	"os"
)

//Metadata represents the metadata of a file.
//It contains:
//The name of the file
//The size of the file
//The Metafile is the concatenation of the sha256 of each files
//Chunks is a map used to store the chunks mapped to its hash
//The MetaHash, i.e., the hash of the metafile. The metahash is the only unique identifier of the file.
type Metadata struct {
	Name        string
	Size        uint32
	Metafile    []byte
	Chunks map[string][]byte
	MetaHash    []byte
}

const CHUNK_SIZE = 8192
const PATH_SHAREDFILES = "./_SharedFiles/"

var hasher = sha256.New()

//MetadataFromIndexing builds the metadata for a file and index it.
func MetadataFromIndexing(fileName string) (*Metadata, error) {
	metadata := &Metadata{Name: fileName}
	metadata.Chunks = make(map[string][]byte)

	filePath := PATH_SHAREDFILES + fileName
	file, err := os.Open(filePath)
	helper.LogError(err)
	defer file.Close()

	if err == nil {
		chunk := make([]byte, CHUNK_SIZE)
		metafileSlice := make([]byte, 0, 320)

		for {
			n, err := file.Read(chunk)

			if err == io.EOF{
				break
			}

			if err != nil {
				helper.LogError(err)
				return nil, err
			}

			if n > 0 {
				hash, err := metadata.hash(chunk[:n])

				if err != nil {
					helper.LogError(err)
					return nil, err
				}

				metadata.Chunks[hex.EncodeToString(hash)] = chunk[:n]
				metafileSlice = append(metafileSlice, hash...)
			}

		}

		metadata.Size = uint32(len(metafileSlice))
		metaHash, err := metadata.hash(metafileSlice)
		if err != nil {
			helper.LogError(err)
			return nil, err
		}
		metadata.MetaHash = metaHash
		metadata.Metafile = metafileSlice

		return metadata, nil
	}

	return nil, err
}

//hash returns the hash (as []byte) of the chunk or an error.
func (metadata *Metadata) hash(chunk []byte) ([]byte, error) {
	//hash
	hasher.Reset()
	_, err := hasher.Write(chunk)

	if err != nil {
		return nil, err
	}
	hash := hasher.Sum(nil)
	return hash, nil
}
