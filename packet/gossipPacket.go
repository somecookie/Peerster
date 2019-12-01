package packet

import (
	"github.com/dedis/protobuf"
	"github.com/somecookie/Peerster/helper"
)

//GossipPacket is the only type of message sent between the different nodes
//In a GossipPack, one and only one field should be non-nil
type GossipPacket struct {
	Simple        *SimpleMessage
	Rumor         *RumorMessage
	Status        *StatusPacket
	Private       *PrivateMessage
	DataRequest   *DataRequest
	DataReply     *DataReply
	SearchRequest *SearchRequest
	SearchReply   *SearchReply
	TLCMessage    *TLCMessage
	Ack           *TLCAck
}

//GetPacketBytes serializes the GossipPacket message
func GetPacketBytes(message interface{}) ([]byte, error) {
	packetBytes, err := protobuf.Encode(message)
	if err != nil {
		helper.LogError(err)
		return nil, err
	}
	return packetBytes, nil
}

//GetGossipPacket deserialize the n first bytes of buffer to get a GossipPacket
func GetGossipPacket(buffer []byte, n int) (*GossipPacket, error) {
	receivedPacket := &GossipPacket{}
	err := protobuf.Decode(buffer[:n], receivedPacket)
	if err != nil {
		helper.LogError(err)
		return nil, err
	}
	return receivedPacket, err
}

func (gp *GossipPacket) GetOriginAndID() (string, uint32){
	if gp.Rumor != nil{
		return gp.Rumor.Origin, gp.Rumor.ID
	}else if gp.TLCMessage != nil {
		return gp.TLCMessage.Origin, gp.TLCMessage.ID
	}else{
		return "",0
	}
}