package packet

import (
	"encoding/hex"
	"fmt"
	"github.com/somecookie/Peerster/blockchain"
)

type TLCMessage struct {
	Origin      string
	ID          uint32
	Confirmed   int
	TxBlock     blockchain.BlockPublish
	VectorClock *StatusPacket
	Fitness     float32
}

type TLCAck PrivateMessage

func PrintUnconfirmedMessage(message *TLCMessage) {
	fmt.Printf("UNCONFIRMED GOSSIP origin %s ID %d file name %s size %d metahash %s\n",
		message.Origin, message.ID, message.TxBlock.Transaction.Name, message.TxBlock.Transaction.Size, hex.EncodeToString(message.TxBlock.Transaction.MetafileHash))
}

func PrintSendingTLCAck(ack *TLCAck){
	fmt.Printf("SENDING ACK origin %s ID %d\n", ack.Destination, ack.ID)
}

func PrintConfirmedMessage(message *TLCMessage){
	fmt.Printf("CONFIRMED GOSSIP origin %s ID %d file name %s size %d metahash %s\n",
		message.Origin, message.Confirmed, message.TxBlock.Transaction.Name, message.TxBlock.Transaction.Size, hex.EncodeToString(message.TxBlock.Transaction.MetafileHash))
}
