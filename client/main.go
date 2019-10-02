package main

import (
	"flag"
	"fmt"
	"github.com/rferreir/Peerster/packet"
)

var (
	uiPort string
	msg    string
)

func init() {
	flag.StringVar(&uiPort, "UIPort", "8080", "port for the UI client (default \"8080\")")
	flag.StringVar(&msg, "msg", "", "message to be sent")
	flag.Parse()
}

func main() {
	simpleMessage := packet.SimpleMessage{
		OriginalName:  "",
		RelayPeerAddr: "",
		Contents:      msg,
	}

	fmt.Println(simpleMessage)
}
