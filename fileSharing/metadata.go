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
//The metafile file, i.e., a binary file containing concatenation of the 32-byte SHA-256 of each chunk of the file
//The MetaHash, i.e., the hash of the metafile. The metahash is the only unique identifier of the file.
type Metadata struct {
	Name     string
	Size     uint32
	Metafile []byte
	MetaHash []byte
}

const CHUNK_SIZE = 8192
const PATH_SHAREDFILES = "./_SharedFiles/"

var hasher = sha256.New()

//MetadataFromIndexing builds the metadata for a file and index it.
func MetadataFromIndexing(fileName string) (*Metadata, error) {
	fileMetadata := &Metadata{Name: fileName}
	filePath := PATH_SHAREDFILES + fileName
	file, err := os.Open(filePath)
	helper.LogError(err)
	defer file.Close()

	if err == nil {
		chunk := make([]byte, CHUNK_SIZE)
		metafileSlice := make([]byte, 0, 320)

		for {
			n, err := file.Read(chunk)

			if err == io.EOF {
				break
			}

			if err != nil {
				helper.LogError(err)
				return nil, err
			}

			if n > 0 {
				hash, err := hashAndStore(fileName, chunk[:n], false)

				if err != nil {
					helper.LogError(err)
					return nil, err
				}

				metafileSlice = append(metafileSlice, hash...)
			}

		}

		fileMetadata.Size = uint32(len(metafileSlice))

		metaHash, err := hashAndStore(fileName, metafileSlice, true)

		if err != nil{
			helper.LogError(err)
			return nil, err
		}
		fileMetadata.MetaHash = metaHash
		fileMetadata.Metafile = metafileSlice
		return fileMetadata, nil
	}

	return nil, err
}

//hashAndStore stores the chunk at _SharedFiles/.chunks/fileName/hash_of_chunk
//It returns the hash (as []byte) of the chunk or an error.
//If metafile is true, the name of the stored filed is metafile
func hashAndStore(fileName string, chunk []byte, metafile bool) ([]byte, error) {

	//hash
	hasher.Reset()
	_, err := hasher.Write(chunk)

	if err != nil {
		return nil, err
	}
	hash := hasher.Sum(nil)

	//checks if the directory .chunks exists, if not it creates it
	path := PATH_SHAREDFILES +".chunks"
	if err := mkdirIsNotExist(path); err != nil{
		return nil, err
	}

	//checks if the directory _SharedFiles/.chunks/fileName/ exists, if not it creates it
	path += "/"+fileName
	if err := mkdirIsNotExist(PATH_SHAREDFILES +".chunks/"+fileName); err != nil{
		return nil, err
	}
	//store
	if metafile{
		path += "/metafile"
	}else{
		path += "/"+hex.EncodeToString(hash)
	}

	fw, err := os.Create(path)
	defer fw.Close()

	if err != nil{
		return nil, err
	}

	if _, err := fw.Write(chunk); err != nil{
		return nil, err
	}

	return hash, nil
}


//mkdirIsNotExist checks if the path exists and makes a directory if it is not the case.
func mkdirIsNotExist(path string) error{
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			if err := os.Mkdir(path, 0775); err != nil{
				return err
			}
		} else {
			return err
		}
	}

	return nil
}
