package main

import (
	"flag"
	"github.com/somecookie/Peerster/helper"
	"github.com/somecookie/Peerster/packet"
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
	flag.StringVar(&gossiperAddr, "gossipAddr", "127.0.0.1", "ip address of the gossiper")
	flag.Parse()
	gossiperAddr += ":"+uiPort
}

func main() {
	udpAddr, conn := connectUDP()
	defer conn.Close()
	packetBytes, err := packet.GetPacketBytes(&packet.Message{Text:msg})
	helper.HandleCrashingErr(err)
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

//connectUDP connects to the gossip through UDP.
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
