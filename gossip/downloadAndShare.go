package gossip

import (
	"encoding/hex"
	"fmt"
	"github.com/somecookie/Peerster/fileSharing"
	"github.com/somecookie/Peerster/helper"
	"github.com/somecookie/Peerster/packet"
	"math"
	"net"
	"os"
	"time"
)

const DOWNLOAD_TIMEOUT = 5

//startDownload starts the downloading process.
//The gossiper wants to download the file with metahash *message.Request from
//*message.Destination and store it with the name *message.File.
func (g *Gossiper) startDownload(message *packet.Message) {

	fmt.Printf("DOWNLOADING metafile of %s from %s\n", *message.File, *message.Destination)
	dr := &packet.DataRequest{
		Origin:      g.Name,
		Destination: *message.Destination,
		HopLimit:    9,
		HashValue:   *message.Request,
	}

	newMetadata := &fileSharing.Metadata{
		Name:     *message.File,
		MetaHash: *message.Request,
		Chunks:   make(map[string][]byte),
		Size: 0,
	}

	g.FilesIndex.Store(newMetadata)

	g.DSDV.Mutex.RLock()
	if g.DSDV.Contains(*message.Destination) {
		metaHashStr := hex.EncodeToString(*message.Request)
		nextHopAddr := g.DSDV.NextHop[*message.Destination]
		g.DSDV.Mutex.RUnlock()

		g.sendMessage(&packet.GossipPacket{DataRequest: dr}, nextHopAddr)

		g.Requested.Mutex.Lock()
		if _, ok := g.Requested.State[*message.File]; !ok {
			g.Requested.State[*message.File] = make(map[string]uint64)
		}

		g.Requested.State[*message.File][metaHashStr] = 0
		g.Requested.Mutex.Unlock()


		go g.waitDownloadACK(nextHopAddr, dr, metaHashStr, *message.File)

	} else {
		g.DSDV.Mutex.RUnlock()
		return
	}

}

func (g *Gossiper) waitDownloadACK(nextHopAddr *net.UDPAddr, dataRequest *packet.DataRequest, metaHashStr, name string) {
	from := dataRequest.Destination
	hash := hex.EncodeToString(dataRequest.HashValue)
	ack := make(chan *packet.DataReply)

	g.Requested.Mutex.Lock()
	g.addDownloadACK(from, hash, ack)
	chunkNbr := g.Requested.State[name][metaHashStr]
	g.Requested.Mutex.Unlock()
	ticker := time.NewTicker(DOWNLOAD_TIMEOUT * time.Second)
	for {
		select {
		case <-ticker.C:
			if chunkNbr == 0{
				fmt.Printf("DOWNLOADING metafile of %s from %s\n", name, from)
			}else{
				fmt.Printf("DOWNLOADING %s chunk %d from %s\n", name, chunkNbr-1, from)
			}
			g.sendMessage(&packet.GossipPacket{DataRequest: dataRequest}, nextHopAddr)
		case dataReply := <-ack:
			ticker.Stop()

			g.Requested.Mutex.Lock()
			g.removeDownloadACK(from, hash)
			g.Requested.Mutex.Unlock()
			g.processReply(dataReply, name, metaHashStr)
			return
		}
	}

}

func (g *Gossiper) processReply(dataReply *packet.DataReply,fileName, metaHashStr string) {
	if dataReply.Data != nil && len(dataReply.Data) != 0{

		g.Requested.Mutex.Lock()
		chunkNbr := g.Requested.GetAndIncrement(fileName, metaHashStr)
		g.Requested.Mutex.Unlock()

		g.FilesIndex.Mutex.RLock()
		metadata := g.FilesIndex.Index[metaHashStr]
		g.FilesIndex.Mutex.RUnlock()

		if chunkNbr > 0{
			metadata.LastReceivedChunk = chunkNbr
		}

		if chunkNbr == 0 {
			metadata.Metafile = make([]byte, 0, len(dataReply.Data))
			metadata.Metafile = append(metadata.Metafile, dataReply.Data...)
			metadata.NbrChunks = uint64(math.Ceil(float64(len(dataReply.Data))/32.0))
			dataRequest, nextHopAddr := g.sendNextRequest(metadata, dataReply, chunkNbr+1)
			if dataRequest != nil && nextHopAddr != nil{
				go g.waitDownloadACK(nextHopAddr, dataRequest, metaHashStr, metadata.Name)
			}


		} else if chunkNbr < metadata.NbrChunks {

			metadata.Chunks[hex.EncodeToString(dataReply.HashValue)] = make([]byte, 0, len(dataReply.Data))
			metadata.Chunks[hex.EncodeToString(dataReply.HashValue)] = append(metadata.Chunks[hex.EncodeToString(dataReply.HashValue)], dataReply.Data...)
			dataRequest, nextHopAddr := g.sendNextRequest(metadata, dataReply, chunkNbr+1)

			if dataRequest != nil && nextHopAddr != nil{
				go g.waitDownloadACK(nextHopAddr, dataRequest, metaHashStr, metadata.Name)
			}
		} else {
			metadata.Chunks[hex.EncodeToString(dataReply.HashValue)] = make([]byte, 0, len(dataReply.Data))
			metadata.Chunks[hex.EncodeToString(dataReply.HashValue)] = append(metadata.Chunks[hex.EncodeToString(dataReply.HashValue)], dataReply.Data...)
			g.reconstructFile(metadata)
		}

	}

}

func (g *Gossiper) sendNextRequest(metadata *fileSharing.Metadata, dataReply *packet.DataReply, nextChunk uint64) (*packet.DataRequest, *net.UDPAddr) {

	var dest string

	if owners := g.Matches.GetOwners(metadata.Name, hex.EncodeToString(metadata.MetaHash), nextChunk); owners != nil{
		dest = owners[0]
		fmt.Printf("DOWNLOADING %s chunk %d from %s\n", metadata.Name, nextChunk, dest)
	}else{
		return nil,nil
	}

	nextHashValue := metadata.Metafile[(nextChunk-1)*32 : nextChunk*32]
	dataRequest := &packet.DataRequest{
		Origin:      g.Name,
		Destination: dest,
		HopLimit:    9,
		HashValue:   nextHashValue,
	}
	g.DSDV.Mutex.RLock()
	nextHopAddr := g.DSDV.NextHop[dataRequest.Destination]
	g.DSDV.Mutex.RUnlock()
	g.sendMessage(&packet.GossipPacket{DataRequest: dataRequest}, nextHopAddr)
	return dataRequest, nextHopAddr
}

func (g *Gossiper) reconstructFile(metadata *fileSharing.Metadata) {

	fsf, err := os.Create(fileSharing.PATH_SHAREDFILES+metadata.Name)
	defer fsf.Close()
	if err != nil{
		helper.LogError(err)
		return
	}

	fd, err := os.Create(fileSharing.PATH_DOWNLOADS+metadata.Name)
	defer fd.Close()
	if err != nil{
		helper.LogError(err)
		return
	}

	for i := uint64(0); i < metadata.NbrChunks; i++{
		hashChunk := metadata.Metafile[i*32:(i+1)*32]
		chunk := metadata.Chunks[hex.EncodeToString(hashChunk)]
		metadata.Size += uint32(len(chunk))

		if _, err:= fsf.Write(chunk); err != nil{
			helper.LogError(err)
			return
		}
		if _, err:= fd.Write(chunk); err != nil{
			helper.LogError(err)
			return
		}

	}

	fmt.Printf("RECONSTRUCTED file %s\n", metadata.Name)
}

func (g *Gossiper) addDownloadACK(from, hash string, ack chan *packet.DataReply) {
	if _, ok := g.Requested.ACKs[from]; !ok {
		g.Requested.ACKs[from] = make(map[string]chan *packet.DataReply)
	}
	g.Requested.ACKs[from][hash] = ack

}

func (g *Gossiper) removeDownloadACK(from, hash string) {
	if _, ok := g.Requested.ACKs[from]; ok {
		if c, ok := g.Requested.ACKs[from][hash]; ok {
			close(c)
			delete(g.Requested.ACKs[from], hash)
		}
	}
}

//IndexFile indexes the file at _SharedFiles called.
//It creates the metadata of the file and stores it in the sync.Map called FilesIndex.
func (g *Gossiper) IndexFile(fileName string) {
	metadata, err := fileSharing.MetadataFromIndexing(fileName)
	helper.LogError(err)
	if err == nil {
		g.FilesIndex.Mutex.Lock()
		g.FilesIndex.Store(metadata)
		g.FilesIndex.Mutex.Unlock()

		g.TLCMajority.Lock()
		defer g.TLCMajority.Unlock()

		if g.ackAll{
			g.BroadcastNewFile(metadata)
		}else if !g.TLCMajority.ReicvCommand{
			g.TLCMajority.ReicvCommand = true
			g.BroadcastNewFile(metadata)
			g.TryNextRound()
		}else{
			g.TLCMajority.Queue.Enqueue(metadata)
		}



	}
}

