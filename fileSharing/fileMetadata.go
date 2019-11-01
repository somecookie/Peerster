package fileSharing

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/somecookie/Peerster/helper"
	"io"
	"os"
)

//FileMetadata represents the metadata of a file.
//It contains:
//The name of the file
//The size of the file
//The metafile file, i.e., a binary file containing concatenation of the 32-byte SHA-256 of each chunk of the file
//The MetaHash, i.e., the hash of the metafile. The metahash is the only unique identifier of the file.
type FileMetadata struct {
	Name     string
	Size     uint32
	Metafile *os.File
	MetaHash string
}

const CHUNK_SIZE = 8192

//FileMetadataFactory builds the metadata for a file.
//It may return nil if an error occurred during the creation.
func FileMetadataFactory(fileName string) *FileMetadata{
	fileMetadata := &FileMetadata{Name:fileName}
	cwd, err := os.Getwd()
	helper.LogError(err)

	if err == nil{
		filePath := cwd+"/_SharedFiles/"+fileName
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
					return nil
				}

				if n > 0 {
					hashFunction := sha256.New()
					hashFunction.Write(chunk[:n])
					hash := hashFunction.Sum(nil)
					metafileSlice = append(metafileSlice, hash...)
				}

			}

			fileMetadata.Size = uint32(len(metafileSlice))

			hashFunction := sha256.New()
			hashFunction.Write(metafileSlice)
			metahash := hex.EncodeToString(hashFunction.Sum(nil))
			fileMetadata.MetaHash = metahash

			metafile, err := os.Create(cwd + "/_SharedFiles/" + metahash + ".bin")
			helper.LogError(err)

			if _, err := metafile.Write(metafileSlice); err != nil {
				helper.LogError(err)
				return nil
			}

			fileMetadata.Metafile = metafile
			return fileMetadata
		}
	}


	return nil
}
