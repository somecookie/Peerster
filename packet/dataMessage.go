package packet

//DataRequest represents the messages for the chunk and metafile requests.
//Origin (string) contains the identifier of the node sending the packet.
//Destination (string) is the destination node identifier of the message
//HopLimit (uint32) denotes the number of nodes that can be reached before the message is discarded.
//HashValue ([]byte) is either the hash of the requested chunk or the MetaHash if the request is for a metafile.
type DataRequest struct {
	Origin      string
	Destination string
	HopLimit    uint32
	HashValue   []byte
}

//DataRequest represents the messages for the chunk and metafile replies.
//Origin (string) contains the identifier of the node sending the packet.
//Destination (string) is the destination node identifier of the message
//HopLimit (uint32) denotes the number of nodes that can be reached before the message is discarded.
//HashValue ([]byte) is either the hash of the requested chunk or the MetaHash if the request is for a metafile.
//Data ([]byte) is the actual data (metafile or chunk)
type DataReply struct {
	Origin      string
	Destination string
	HopLimit    uint32
	HashValue   []byte
	Data        []byte
}
