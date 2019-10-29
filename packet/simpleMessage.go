package packet

import "fmt"

type SimpleMessage struct {
	OriginalName  string
	RelayPeerAddr string
	Contents      string
}

func PrintSimpleMessage(message *SimpleMessage) {
	fmt.Printf("SIMPLE MESSAGE origin %s from %s contents %s\n", message.OriginalName, message.RelayPeerAddr, message.Contents)
}

