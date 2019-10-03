package main

import (
	"Peerster/helper"
	"Peerster/packet"
	"flag"
	"github.com/dedis/protobuf"
	"log"
	"net"
)

var (
	uiPort       string
	msg          string
	gossiperAddr string
	clientAddr   string
)

func init() {
	flag.StringVar(&uiPort, "UIPort", "8080", "port for the UI client (default \"8080\")")
	flag.StringVar(&msg, "msg", "", "message to be sent")
	flag.Parse()
	gossiperAddr = "127.0.0.1:" + uiPort
}

func main() {

	udpAddr, conn := connectUDP()
	defer conn.Close()
	packetBytes := getPacketBytes()
	sendPacket(conn, packetBytes, udpAddr)
}

// sendPacket sends the previously created packet.
func sendPacket(conn *net.UDPConn, packetBytes []byte, udpAddr *net.UDPAddr) {
	i, err := conn.Write(packetBytes)
	if err != nil {
		helper.HandleCrashingErr(err)
	} else if len(packetBytes) != i {
		log.Printf("%d bytes have been sent instead of %d\n", i, len(packetBytes))
	}
}

// getPacketBytes create and serialize the SimpleMessage that will be sent by the client.
func getPacketBytes() []byte {
	simpleMessage := &packet.SimpleMessage{
		OriginalName:  "",
		RelayPeerAddr: "",
		Contents:      msg,
	}
	packetBytes, err := protobuf.Encode(simpleMessage)
	if err != nil {
		helper.HandleCrashingErr(err)
	}

	return packetBytes
}

//connectUDP connects to the gossiper through UDP.
// It returns the resolved address used for UDP and the connection.
func connectUDP() (*net.UDPAddr, *net.UDPConn) {
	udpAddr, err := net.ResolveUDPAddr("udp4", gossiperAddr)
	if err != nil {
		helper.HandleCrashingErr(err)
	}
	conn, err := net.DialUDP("udp4", nil, udpAddr)
	if err != nil {
		helper.HandleCrashingErr(err)
	}
	return udpAddr, conn
}
