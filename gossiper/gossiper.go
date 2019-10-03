package gossiper

import (
	"Peerster/helper"
	"Peerster/packet"
	"fmt"
	"github.com/dedis/protobuf"
	"log"
	"net"
)

type Gossiper struct {
	name          string
	peers         []*net.UDPAddr
	simple        bool
	addressClient *net.UDPAddr
	addressGossip *net.UDPAddr
}


//BasicGossiperFactory creates a Gossiper from the parsed flags of main.go.
func BasicGossiperFactory(uiPort, gossipPort, ip, name string, peers []*net.UDPAddr, simple bool) (*Gossiper, error) {
	udpAddrClient, err := net.ResolveUDPAddr("udp4", ip+":"+uiPort)
	if err != nil {
		return nil, err
	}

	udpAddrGossip, err := net.ResolveUDPAddr("udp4", ip+":"+gossipPort)
	if err != nil {
		return nil, err
	}

	return &Gossiper{
		name:          name,
		peers:         peers,
		simple:        simple,
		addressClient: udpAddrClient,
		addressGossip: udpAddrGossip,
	}, nil
}

func (g *Gossiper) HandleUDPClient(){
	conn, err := net.ListenUDP("udp4", g.addressClient)

	if err != nil{
		helper.HandleCrashingErr(err)
	}

	defer conn.Close()

	buffer := make([]byte, 1024)
	for{
		n,_,err := conn.ReadFromUDP(buffer)

		if err != nil{
			log.Println(err)
		}else{
			message := &packet.SimpleMessage{}
			err := protobuf.Decode(buffer[:n], message)

			if err != nil{
				log.Println(err)
			}else{
				g.handleClientSimpleMessage(message)
			}
		}
	}

}

func (g *Gossiper) handleClientSimpleMessage(message *packet.SimpleMessage){
	fmt.Printf("CLIENT MESSAGE %s\n", message.Contents)
}
