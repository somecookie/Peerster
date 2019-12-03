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
	dest          string
	file          string
	requestString string
	keywords      string
	budget        uint64
)

func init() {
	flag.StringVar(&uiPort, "UIPort", "8080", "port for the UI client (default \"8080\")")
	flag.StringVar(&msg, "msg", "", "message to be sent: if the -dest flag is present, this is a private message, otherwise it's a rumor message")
	flag.StringVar(&dest, "dest", "", "destination for the private message; can be omitted")
	flag.StringVar(&file, "file", "", "file to be indexed by the gossiper")
	flag.StringVar(&requestString, "request", "", "request a chunk or metafile of this hash")
	flag.StringVar(&keywords, "keywords", "", "comma separated list of searched keywords")
	flag.Uint64Var(&budget, "budget", 0, "budget for the search")

	flag.Parse()
}

func validFlags() bool {

	if file != "" && requestString != "" && msg == "" && keywords == "" && budget == 0 {
		return true //download
	} else if dest != "" && file == "" && requestString == "" && msg != "" && keywords == "" && budget == 0{
		return true //private message
	} else if dest == "" && file != "" && requestString == "" && msg == "" && keywords == "" && budget == 0{
		return true //file sharing
	} else if dest == "" && file == "" && requestString == "" && msg != ""&& keywords == "" && budget == 0 {
		return true //rumor message
	} else if dest == "" && file == "" && requestString == "" && msg == "" && keywords != "" {
		return true //search
	} else {
		return false
	}
}

func main() {

	if !validFlags() {
		fmt.Println("ERROR (Bad argument combination)")
		os.Exit(1)
	}

	udpAddr, conn := connectUDP()
	defer conn.Close()

	msg := &packet.Message{
		Text: msg,
	}

	if requestString != "" {
		request, err := hex.DecodeString(requestString)
		if err != nil || len(request) != 32 {
			//os.Stderr.WriteString("ERROR (Unable to decode hex hash)")
			fmt.Println("ERROR (Unable to decode hex hash)")
			os.Exit(1)
		}

		msg.Request = &request
	}

	if dest != "" {
		msg.Destination = &dest
	}

	if file != "" {
		msg.File = &file
	}

	if keywords != "" {
		msg.Keywords = &keywords
	}

	if budget > 0 {
		msg.Budget = &budget
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
	udpAddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:"+uiPort)
	if err != nil {
		helper.HandleCrashingErr(err)
	}
	conn, err := net.DialUDP("udp4", nil, udpAddr)
	if err != nil {
		helper.HandleCrashingErr(err)
	}
	return udpAddr, conn
}
