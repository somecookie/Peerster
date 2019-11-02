package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/somecookie/Peerster/helper"
	"github.com/somecookie/Peerster/packet"
	"log"
	"net"
	"os"
)

var (
	uiPort        string
	msg           string
	gossiperAddr  string
	clientAddr    string
	dest          string
	fileName      string
	requestString string
)

func init() {
	flag.StringVar(&uiPort, "UIPort", "8080", "port for the UI client (default \"8080\")")
	flag.StringVar(&msg, "msg", "", "message to be sent: if the -dest flag is present, this is a private message, otherwise it's a rumor message")
	flag.StringVar(&dest, "dest", "", "destination for the private message; can be omitted")
	flag.StringVar(&gossiperAddr, "gossipAddr", "127.0.0.1", "ip address of the gossiper")
	flag.StringVar(&fileName, "file", "", "file to be indexed by the gossiper")
	flag.StringVar(&requestString, "request", "", "requestString a chunk or metafile of this hash")
	flag.Parse()
	gossiperAddr += ":" + uiPort
}

func main() {
	udpAddr, conn := connectUDP()
	defer conn.Close()

	msg := &packet.Message{
		Text:msg,
	}

	if requestString != ""{
		request, err := hex.DecodeString(requestString)
		if err != nil || len(request) != 32{
			//os.Stderr.WriteString("ERROR (Unable to decode hex hash)")
			fmt.Println("ERROR (Unable to decode hex hash)")
			os.Exit(1)
		}

		msg.Request = &request
	}

	if dest != "" {
		msg.Destination = &dest
	}

	if fileName != ""{
		msg.File = &fileName
	}

	packetBytes, err := packet.GetPacketBytes(msg)

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
