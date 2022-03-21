package main

// Lacking a func to compare and update the nodeMaps
/* For now, the non-seed nodes will periodically communicate with the seed nodes to update node map.
Non-seed nodes will not communicate with each other. Can implement this later –>
randomly select a node in node map to communicate; when starting out, non-seed comm with seed,
seed will comm with another seed.
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
	// Initialising nodeMap with seed nodes
	g.mu.Lock()
	for _, str := range seedNodesArr {
		node := node{membership: true, containerName: nodeidToContainerName(str)}
		g.nodeMap[str] = node
		// Populating seedNode map as well
		seedNodesMap[str] = node

	}
	// Periodically select a random seed node to exchange data
	ticker := time.NewTicker(10 * time.Second)
	for range ticker.C {
		seedID := seedNodesArr[rand.Intn(len(seedNodesArr))]
		seedNode := seedNodesMap[seedID]
		con, err := net.Dial(CONN_TYPE, seedNode.containerName+CONN_PORT)
		checkErr(err)
		localNode := g.nodeMap[getLocalNodeID()]
		enc := gob.NewEncoder(con)
		errEnc := enc.Encode(localNode)
		checkErr(errEnc)
		con.Close()
	}
}

func (g *gossip) listenMsg(con net.Conn) {
	dec := gob.NewDecoder(con)
	var newNode node
	err := dec.Decode(&newNode)
	checkErr(err)
	fmt.Println(newNode)
	con.Close()
}

// Helper functions
func getLocalContainerName() string {
	var output string
	switch os.Getenv("NODE_ID") {
	case "0":
		output = "node0"
	case "1":
		output = "node1"
	case "2":
		output = "node3"
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
