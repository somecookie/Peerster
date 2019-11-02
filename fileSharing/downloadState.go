package fileSharing

import "sync"

//DownloadState is a structure used to keep track of the downloads/uploads
//The State fields maps the origin to a file metahash. Then the metahash is mapped to a slice of bools. Each entry of
//the slice represents the state of the chunk,i.e., if it has been downloaded/uploaded. At 0-index is the metafile.
type DownloadState struct{
	State map[string]map[string][]bool
	Mutex sync.RWMutex
}
