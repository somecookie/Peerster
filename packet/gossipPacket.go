package packet

import (
	"Peerster/helper"
	"github.com/dedis/protobuf"
)

//GossipPacket is the only type of message sent between the different nodes
//In a GossipPack, one and only one field should be non-nil
type GossipPacket struct {
	Simple *SimpleMessage
	Rumor  *RumorMessage
	Status *StatusPacket
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
