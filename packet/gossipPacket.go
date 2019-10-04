package packet

import (
	"Peerster/helper"
	"github.com/dedis/protobuf"
)

type GossipPacket struct {
	Simple *SimpleMessage
}

func GetPacketBytes(message interface{}) ([]byte, error) {

	packetBytes, err := protobuf.Encode(message)
	if err != nil {
		helper.LogError(err)
		return nil, err
	}
	return packetBytes, nil
}