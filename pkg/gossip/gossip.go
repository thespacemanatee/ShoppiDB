package gossip

/*
4. Membership history :) Add/Delete new nodes. snapshot of nodeMap with timestamp can be stored onto the db.
*/
/*
 System design for gossip:
	1. Gossip should have its own http.Client created in a goroutine that deals entirely
	with the client logic i.e. sending out its node structure periodically.
	2. Server: create a function in gossip that starts the http.Server. Main() will call the
	gossip function which will call the http_api function.
	3. Server should still send back the feedback for the client to update its gossip.CommNodeMap and gossip.VirtualNodeMap.

*/
// handler should be able to return what it received, but it does not return anything. Need to figure out how to get the value.
// 1. Global variable 2. Read handler doc 3. Google/stackoverflow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	CONN_PORT  = ":8080"
	CONN_TYPE  = "http://"
	HTTP_ROUTE = "/gossip"
)

type GossipMessage struct {
	Update        bool
	ContainerName string
	MyCommNodeMap map[string]GossipNode
	// MyVirtualNodeMap map[[2]string]GossipNode //Itoa: convert int to string first when populating. Convert to int when reading.
}

type GossipNode struct {
	ContainerName string
	TokenSet      [][]int
	Membership    bool
}

type Gossip struct {
	mu          sync.Mutex
	CommNodeMap map[string]GossipNode //Map ID to physical node
	// VirtualNodeMap map[[2]string]GossipNode //Itoa: convert int to string first when populating. Convert to int when reading.
	HttpClient *http.Client
}

/*
CLIENT
*/

func (g *Gossip) Start() {
	// get seed nodes []string
	seedNodesArr := getSeedNodes()
	// if node is seednode, sleep for a min before communicating with other nodes
	isSeedNode := false
	for _, nodeID := range seedNodesArr {
		if GetLocalNodeID() == nodeID {
			isSeedNode = true
			fmt.Println("I am seed node!")
			time.Sleep(time.Minute)
		}
	}
	// Updates g.nodeMap with seed nodes and return a fresh seedNodesMap
	seedNodesMap := g.populate(seedNodesArr)

	// Periodically select a random seed node to exchange data
	nodeIDInt, err := strconv.Atoi(GetLocalNodeID())
	checkErr(err)
	ticker := time.NewTicker(time.Duration(2*nodeIDInt+5) * time.Second)
	timer := time.NewTimer(time.Minute)

	//Only non-seed nodes execute the below block of code
	if !isSeedNode {
	loop1:
		for {
			select {
			case <-ticker.C:
				// Can consider having this being run on a goroutine, but implement it such that the client don't dial to the same node successively in a row
				seedID := seedNodesArr[rand.Intn(len(seedNodesArr))]
				seedNode := seedNodesMap[seedID]
				target := CONN_TYPE + seedNode.ContainerName + CONN_PORT + HTTP_ROUTE
				g.clientSendMsgWithHTTP(g.HttpClient, target)
			case <-timer.C:
				ticker.Stop()
				fmt.Println("")
				fmt.Println("PARTY TIME")
				fmt.Println("")
				break loop1
			}
		}
	} else {
		fmt.Println("")
		fmt.Println("SEED NODE PARTY TIME!")
		fmt.Println("")
	}

	ticker2 := time.NewTicker(10 * time.Second)
	for range ticker2.C {
		randNode := g.getRandNode()
		target := CONN_TYPE + randNode.ContainerName + CONN_PORT + HTTP_ROUTE
		g.clientSendMsgWithHTTP(g.HttpClient, target)
	}
}

// Helper functions for gossip.Start

func (g *Gossip) clientSendMsgWithHTTP(client *http.Client, target string) {
	msg := GossipMessage{ContainerName: GetLocalContainerName(), MyCommNodeMap: g.CommNodeMap}
	msgJson, err1 := json.Marshal(msg)
	checkErr(err1)
	req, err2 := http.NewRequest(http.MethodPost, target, bytes.NewBuffer(msgJson))
	checkErr(err2)
	req.Header.Set("Content-Type", "application/json")
	resp, err3 := client.Do(req)
	checkErr(err3)
	defer resp.Body.Close()
	var respMsg GossipMessage
	json.NewDecoder(resp.Body).Decode(&respMsg)
	fmt.Println(GetLocalContainerName(), "received", respMsg.MyCommNodeMap, "from", respMsg.ContainerName, "with", respMsg.Update)
	g.recvGossipMsg(respMsg)
}

// Take a string array of seed node IDs and populate them into Gossip.CommNodeMap and return a seedNodeMap map[seedNodeID(string)]GossipNode
func (g *Gossip) populate(seedNodesArray []string) map[string]GossipNode {
	g.mu.Lock()
	seedNodesMap := make(map[string]GossipNode)
	for _, str := range seedNodesArray {
		node := GossipNode{Membership: true, ContainerName: nodeidToContainerName(str)}
		g.CommNodeMap[str] = node
		// Populating seedNode map as well
		seedNodesMap[str] = node
	}
	g.mu.Unlock()
	return seedNodesMap
}

func (g *Gossip) getRandNode() GossipNode {
	g.mu.Lock()
	var randNode GossipNode
	var randomInt int
loop:
	for {
		randomInt = rand.Intn(len(g.CommNodeMap))

		fmt.Println("string(randomInt):", strconv.Itoa(randomInt))
		randNode = g.CommNodeMap[strconv.Itoa(randomInt)]
		if randNode.ContainerName != GetLocalContainerName() {
			fmt.Println(randNode.ContainerName, GetLocalContainerName())
			break loop
		}
	}
	g.mu.Unlock()
	return randNode
}

func (g *Gossip) recvGossipMsg(msg GossipMessage) {
	g.mu.Lock()
	fmt.Println(GetLocalContainerName()+" has received", msg)
	if msg.Update {
		for key, value := range msg.MyCommNodeMap {
			g.CommNodeMap[key] = value
		}
	}
	fmt.Println(GetLocalContainerName(), "has", g.CommNodeMap)
	g.mu.Unlock()
}

/*
SERVER
*/

//Helper functions for gossip.serverStart

func (g *Gossip) CompareAndUpdate(msg GossipMessage) GossipMessage {
	g.mu.Lock()
	fmt.Println("Received GossipMessage from", msg.ContainerName)
	fmt.Println("myNodeMap:", g.CommNodeMap)
	var updateForSender bool
	nodeMapForSender := make(map[string]GossipNode)
	commonNodeCounter := 0
	senderUniqueNodeCounter := 0

	for senderKey, senderValue := range msg.MyCommNodeMap {
		if _, found := g.CommNodeMap[senderKey]; !found {
			// local nodeMap does not contain the node in sender's nodeMap
			g.CommNodeMap[senderKey] = senderValue
			senderUniqueNodeCounter += 1
		} else {
			commonNodeCounter += 1
		}
	}

	if iGotUniqueNodes := len(g.CommNodeMap) - commonNodeCounter - senderUniqueNodeCounter; iGotUniqueNodes > 0 {
		fmt.Println(GetLocalContainerName(), "server has unique nodes!")
		// fmt.Println(updateForSender, "before")
		updateForSender = true
		// fmt.Println(updateForSender, "after")
		for myKey, myValue := range g.CommNodeMap {
			if _, found := msg.MyCommNodeMap[myKey]; !found {
				nodeMapForSender[myKey] = myValue
			}
		}
	} else {
		// fmt.Println(updateForSender, "after")
		updateForSender = false
		fmt.Println("No unique nodes in server!")
	}
	g.mu.Unlock()
	// fmt.Println(updateForSender, "after2")
	gossipMessage := GossipMessage{Update: updateForSender, ContainerName: GetLocalContainerName(), MyCommNodeMap: nodeMapForSender}
	return gossipMessage
}

//  General helper functions
func GetLocalContainerName() string {
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
	case "5":
		output = "node5"
	case "6":
		output = "node6"
	case "7":
		output = "node7"
	case "8":
		output = "node8"
	case "9":
		output = "node9"
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
	case "5":
		containerName = "node5"
	case "6":
		containerName = "node6"
	case "7":
		containerName = "node7"
	case "8":
		containerName = "node8"
	case "9":
		containerName = "node9"
	}
	return containerName
}

func GetMembership() bool {
	var output bool
	switch os.Getenv("MEMBERSHIP") {
	case "yes":
		output = true
	case "no":
		output = false
	}
	return output
}

func GetLocalNodeID() string {
	return os.Getenv("NODE_ID")
}

func checkErr(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

// func deepCopyMap(originalMap map[string]Node) map[string]Node {
// 	newMap := make(map[string]Node)
// 	for key, value := range originalMap {
// 		newMap[key] = value
// 	}
// 	return newMap
// }
