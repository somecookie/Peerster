#!/bin/bash

echo "creating 10 nodes"
rm -r out
mkdir out
./Peerster -name=NodeA -peers=127.0.0.1:5004,127.0.0.1:5005 > out/A.out & 
./Peerster -UIPort=8081 -gossipAddr=127.0.0.1:5001 -name=NodeB -peers=127.0.0.1:5006,127.0.0.1:5005 > out/B.out &
./Peerster -UIPort=8082 -gossipAddr=127.0.0.1:5002 -name=NodeC -peers=127.0.0.1:5006,127.0.0.1:5001 > out/C.out &
./Peerster -UIPort=8083 -gossipAddr=127.0.0.1:5003 -name=NodeD -peers=127.0.0.1:5001,127.0.0.1:5007 > out/D.out &
./Peerster -UIPort=8084 -gossipAddr=127.0.0.1:5004 -name=NodeE -peers=127.0.0.1:5000,127.0.0.1:5002 > out/E.out &
./Peerster -UIPort=8085 -gossipAddr=127.0.0.1:5005 -name=NodeF -peers=127.0.0.1:5008,127.0.0.1:5003 > out/F.out &
./Peerster -UIPort=8086 -gossipAddr=127.0.0.1:5006 -name=NodeG -peers=127.0.0.1:5004,127.0.0.1:5000 > out/G.out &
./Peerster -UIPort=8087 -gossipAddr=127.0.0.1:5007 -name=NodeH -peers=127.0.0.1:5001,127.0.0.1:5002 > out/H.out &
./Peerster -UIPort=8088 -gossipAddr=127.0.0.1:5008 -name=NodeI -peers=127.0.0.1:5009,127.0.0.1:5004 > out/I.out &
./Peerster -UIPort=8089 -gossipAddr=127.0.0.1:5009 -name=NodeJ -peers=127.0.0.1:5005,127.0.0.1:5007 > out/J.out &

echo "Sending 50 times hello"

cd client/
echo $pwd
for i in {0..50}
do
	uiport=(8080 + $i%10)
	./client -msg="Hello $i" -UIPort=$uiport
done

sleep 30
killall Peerster

