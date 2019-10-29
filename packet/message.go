package packet

import (
	"fmt"
	"github.com/dedis/protobuf"
	"github.com/somecookie/Peerster/helper"
)

type Message struct {
	Text string
}

//GetMessage deserialize the n first bytes of buffer to get a GetMessage
func GetMessage(buffer []byte, n int) (*Message, error) {
	receivedPacket := &Message{}
	err := protobuf.Decode(buffer[:n], receivedPacket)
	if err != nil {
		helper.LogError(err)
		return nil, err
	}
	return receivedPacket, err
}

func PrintClientMessage(message *Message) {
	fmt.Printf("CLIENT MESSAGE %s\n", message.Text)
}

