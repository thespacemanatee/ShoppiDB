package main

// Lacking a func to compare and update the nodeMaps
/* For now, the non-seed nodes will periodically communicate with the seed nodes to update node map.
Non-seed nodes will not communicate with each other. Can implement this later –>
randomly select a node in node map to communicate; when starting out, non-seed comm with seed,
seed will comm with another seed.
*/
/*
1. Commit and merge branch
2. Add logic to compare and update node map
	- sender sends its node map, receiver will compare, add nodes that it does not have to its nodeMap, and sends nodes that sender does not have to sender in the form
	of a nodeMap as well.
	- check if a certain key-value pair is in the node map.
3. Add logic to randomly select node in node map to comm
4. Membership history :) Add/Delete new nodes. snapshot of nodeMap with timestamp can be stored onto the db.
*/

import (
	"encoding/gob"
	"fmt"
	"math/rand"
	"net"
	"os"
	"runtime"
	"strings"
	"time"
)

const (
	CONN_PORT = ":8080"
	CONN_TYPE = "tcp"
)

func (g *gossip) clientStart() {
	// get seed nodes []string
	seedNodesArr := getSeedNodes()
	// if node is seednode, don't need to send to itself.
	for _, nodeID := range seedNodesArr {
		if getLocalNodeID() == nodeID {
			runtime.Goexit()
		}
	}
	// seedNode map
	seedNodesMap := make(map[string]node)
	// Updating nodeMap with seed nodes
	for _, str := range seedNodesArr {
		node := node{Membership: true, ContainerName: nodeidToContainerName(str)}
		g.nodeMap[str] = node
		// Populating seedNode map as well
		seedNodesMap[str] = node
	}
	// Periodically select a random seed node to exchange data
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		// Can consider having this being run on a goroutine, but implement it such that the client don't dial to the same node successively in a row
		seedID := seedNodesArr[rand.Intn(len(seedNodesArr))]
		seedNode := seedNodesMap[seedID]
		con, err := net.Dial(CONN_TYPE, seedNode.ContainerName+CONN_PORT)
		checkErr(err)
		g.sendMyNodeMap(con)
		g.waitForResponse(con)
	}
}

// Helper functions for gossip.clientStart
func (g *gossip) waitForResponse(con net.Conn) {
	response := make([]byte, 1024)
	fmt.Println("Waiting for server's response")
	msgLen, errResp := con.Read(response)
	checkErr(errResp)
	// reply := string(response[:msgLen])
	fmt.Println("Server's response is:", string(response[:msgLen]))
	if string(response[:msgLen]) == "no" {
		con.Close()
	} else {
		g.recvNodes(con)
	}
}

func (g *gossip) recvNodes(con net.Conn) {
	g.mu.Lock()
	dec := gob.NewDecoder(con)
	var incNodeMap map[string]node
	err := dec.Decode(&incNodeMap)
	checkErr(err)
	fmt.Println(getLocalContainerName()+" has received", incNodeMap)

	for key, value := range incNodeMap {
		g.nodeMap[key] = value
	}
	g.mu.Unlock()
	con.Close()
}

func (g *gossip) sendMyNodeMap(con net.Conn) {
	// localNode := g.nodeMap[getLocalNodeID()]
	myNodeMap := deepCopyMap(g.nodeMap)
	// myNodeMap := g.nodeMap //pass by value
	enc := gob.NewEncoder(con)
	errEnc := enc.Encode(myNodeMap)
	checkErr(errEnc)
	fmt.Println(getLocalContainerName()+" has sent", myNodeMap)
}

func (g *gossip) serverStart() {
	fmt.Println("Starting server...")
	dataStream, err := net.Listen(CONN_TYPE, CONN_PORT)
	checkErr(err)
	defer dataStream.Close()

	for {
		con, err := dataStream.Accept()
		checkErr(err)
		go g.listenMsg(con)
	}

}

//Helper functions for gossip.serverStart
func (g *gossip) listenMsg(con net.Conn) {
	dec := gob.NewDecoder(con)
	var senderNodeMap map[string]node
	err := dec.Decode(&senderNodeMap)
	checkErr(err)
	fmt.Println(getLocalContainerName()+" has received", senderNodeMap)
	updateForSender, returningNodeMap := g.compareAndUpdate(senderNodeMap)
	sendMsg(con, updateForSender)
	if updateForSender == "yes" {
		sendUpdateNodeMap(con, returningNodeMap)
	}
}

func (g *gossip) compareAndUpdate(senderNodeMap map[string]node) (string, map[string]node) {
	g.mu.Lock()
	fmt.Println("senderNodeMap:", senderNodeMap)
	fmt.Println("myNodeMap:", g.nodeMap)
	updateForSender := "no"
	nodeMapForSender := make(map[string]node)
	commonNodeCounter := 0
	senderUniqueNodeCounter := 0

	for senderKey, senderValue := range senderNodeMap {
		if _, found := g.nodeMap[senderKey]; !found {
			// local nodeMap does not contain the node in sender's nodeMap
			g.nodeMap[senderKey] = senderValue
			senderUniqueNodeCounter += 1
		} else {
			commonNodeCounter += 1
		}
	}
	g.mu.Unlock()
	if iGotUniqueNodes := len(g.nodeMap) - commonNodeCounter - senderUniqueNodeCounter; iGotUniqueNodes > 0 {
		fmt.Println("Server has unique nodes!")
		updateForSender = "yes"
		for myKey, myValue := range g.nodeMap {
			if _, found := senderNodeMap[myKey]; !found {
				nodeMapForSender[myKey] = myValue
			}
		}
	}
	fmt.Println("No unique nodes in server!")
	return updateForSender, nodeMapForSender
}

func sendMsg(con net.Conn, msg string) {
	// responseToSender := make([]byte, 1024)
	fmt.Println("Writing response to sender")
	_, err := con.Write([]byte(msg))
	checkErr(err)
}

func sendUpdateNodeMap(con net.Conn, nodeMap map[string]node) {
	enc := gob.NewEncoder(con)
	err := enc.Encode(nodeMap)
	checkErr(err)
	fmt.Println("Server has sent update node map to client!")
}

//  General helper functions
func getLocalContainerName() string {
	var output string
	switch os.Getenv("NODE_ID") {
	case "0":
		output = "node0"
	case "1":
		output = "node1"
	case "2":
		output = "node2"
	case "3":
		output = "node3"
	case "4":
		output = "node4"
	}
	return output
}

func getSeedNodes() []string {
	return strings.Split(os.Getenv("SEEDNODES"), " ")
}

func nodeidToContainerName(nodeid string) string {
	var containerName string
	switch nodeid {
	case "0":
		containerName = "node0"
	case "1":
		containerName = "node1"
	case "2":
		containerName = "node2"
	case "3":
		containerName = "node3"
	case "4":
		containerName = "node4"
	}
	return containerName
}

func getMembership() bool {
	var output bool
	switch os.Getenv("MEMBERSHIP") {
	case "yes":
		output = true
	case "no":
		output = false
	}
	return output
}

func getLocalNodeID() string {
	return os.Getenv("NODE_ID")
}

func checkErr(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

func deepCopyMap(originalMap map[string]node) map[string]node {
	newMap := make(map[string]node)
	for key, value := range originalMap {
		newMap[key] = value
	}
	return newMap
}
