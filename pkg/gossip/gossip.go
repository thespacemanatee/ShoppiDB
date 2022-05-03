package gossip

/*
4. Membership history :) Add/Delete new nodes. snapshot of nodeMap with timestamp can be stored onto the db.
*/

import (
	httpClient "ShoppiDB/pkg/httpClient"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	Id            int
	ContainerName string
	TokenSet      [][2]int
	Membership    bool
	StatusDown    bool
	failCount     int
}

type Gossip struct {
	mu             sync.Mutex
	CommNodeMap    map[string]GossipNode //Map ID to physical node
	VirtualNodeMap map[[2]int]GossipNode //Itoa: convert int to string first when populating. Convert to int when reading.
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
				fmt.Println("Created new HTTP Client")
				g.clientSendMsgWithHTTP(target, seedNode.ContainerName)
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
		fmt.Println("Created new HTTP Client")
		go g.clientSendMsgWithHTTP(target, randNode.ContainerName)
	}
}

// Helper functions for gossip.Start

func (g *Gossip) clientSendMsgWithHTTP(target string, containerName string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Panic Occur, process recovered", r)
			g.mu.Lock()
			for i, node := range g.CommNodeMap {
				if node.ContainerName == containerName && !node.StatusDown {
					tempNode := node
					tempNode.failCount += 1
					if tempNode.failCount >= 3 {
						tempNode.StatusDown = true
						tempNode.failCount = 0
					}
					g.CommNodeMap[i] = tempNode
				}
			}
			g.mu.Unlock()
		}
	}()
	g.mu.Lock()
	msg := GossipMessage{ContainerName: GetLocalContainerName(), MyCommNodeMap: g.CommNodeMap}
	g.mu.Unlock()
	client := httpClient.GetHTTPClient()
	msgJson, err1 := json.Marshal(msg)
	checkErr(err1)
	req, err2 := http.NewRequest(http.MethodPost, target, bytes.NewBuffer(msgJson))
	checkErr(err2)
	req.Header.Set("Content-Type", "application/json")
	resp, err3 := client.Do(req)
	checkErr(err3)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			checkErr(err)
			return
		}
	}(resp.Body)
	var respMsg GossipMessage
	err := json.NewDecoder(resp.Body).Decode(&respMsg)
	if err != nil {
		checkErr(err)
		return
	}
	fmt.Println(GetLocalContainerName(), "received", respMsg.MyCommNodeMap, "from", respMsg.ContainerName, "with", respMsg.Update)
	g.recvGossipMsg(respMsg)
}

// Take a string array of seed node IDs and populate them into Gossip.CommNodeMap and return a seedNodeMap map[seedNodeID(string)]GossipNode
func (g *Gossip) populate(seedNodesArray []string) map[string]GossipNode {
	g.mu.Lock()
	seedNodesMap := make(map[string]GossipNode)
	for _, str := range seedNodesArray {
		node := GossipNode{Membership: true, ContainerName: nodeidToContainerName(str), StatusDown: false}
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
	lastChar := msg.ContainerName[len(msg.ContainerName)-1:]
	if _, exist := g.CommNodeMap[lastChar]; exist { //May be unnessarcy
		tempNode := g.CommNodeMap[lastChar]
		tempNode.failCount = 0
		tempNode.StatusDown = false
		g.CommNodeMap[lastChar] = tempNode
	}
	if msg.Update {
		for nodeID, gossNode := range msg.MyCommNodeMap {
			if nodeID != GetLocalNodeID() {
				url := CONN_TYPE + gossNode.ContainerName + CONN_PORT + "/checkHeartbeat"
				if _, exist := g.CommNodeMap[nodeID]; exist {
					if g.CommNodeMap[nodeID].StatusDown != gossNode.StatusDown {
						if g.verifyNodeDown(url) == gossNode.StatusDown {
							fmt.Println("Update local copy due to updated status and verfied")
							g.CommNodeMap[nodeID] = gossNode
							for _, rnge := range gossNode.TokenSet {
								g.VirtualNodeMap[rnge] = gossNode
							}
						}
					} else if len(g.CommNodeMap[nodeID].TokenSet) < len(gossNode.TokenSet) { //Have the same status
						g.CommNodeMap[nodeID] = gossNode
						for _, rnge := range gossNode.TokenSet {
							g.VirtualNodeMap[rnge] = gossNode
						}
					}
				} else { //Doesnt exist in the local nodeMap
					g.CommNodeMap[nodeID] = gossNode
					for _, rnge := range gossNode.TokenSet {
						g.VirtualNodeMap[rnge] = gossNode
					}
				}
			}
		}
	}
	fmt.Println(GetLocalContainerName(), "has", g.CommNodeMap)
	fmt.Println("VNodeMap", g.VirtualNodeMap)
	g.mu.Unlock()
}

func (g *Gossip) verifyNodeDown(target string) (result bool) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Panic Occur, process recovered", r)
			fmt.Println("Verified failed connection")
			result = true
		}

	}()
	httpClient := httpClient.GetHTTPClient()
	msg := "check"
	msgJson, err := json.Marshal(msg)
	checkErr(err)
	req, err := http.NewRequest(http.MethodGet, target, bytes.NewBuffer(msgJson))
	checkErr(err)
	resp, err := httpClient.Do(req)
	checkErr(err)
	var respMsg string
	err = json.NewDecoder(resp.Body).Decode(&respMsg)
	if err != nil {
		return true
	}
	fmt.Println("Received checks from", respMsg)
	return false
}

/*
SERVER
*/

//Helper functions for gossip.serverStart

/**
* Return updated GossipMessage after comparing with input GossipMessage
*
*
*
* @param msg The gossip message that is received and compared
*
* @return the updated GossipMessage
 */
func (g *Gossip) CompareAndUpdate(msg GossipMessage) GossipMessage {
	g.mu.Lock()
	fmt.Println("Received GossipMessage from", msg.ContainerName)
	fmt.Println("myNodeMap:", g.CommNodeMap)
	var updateForSender bool
	nodeMapForSender := make(map[string]GossipNode)
	commonNodeCounter := 0
	senderUniqueNodeCounter := 0
	var seedNodeTokenSetIsEmpty bool
	seedNodesArr := getSeedNodes()

	lastChar := msg.ContainerName[len(msg.ContainerName)-1:]
	if _, exist := g.CommNodeMap[lastChar]; exist { //May be unnessarcy
		tempNode := g.CommNodeMap[lastChar]
		tempNode.failCount = 0
		tempNode.StatusDown = false
		g.CommNodeMap[lastChar] = tempNode
	}

	for _, seed := range seedNodesArr {
		if gossNode := msg.MyCommNodeMap[seed]; len(gossNode.TokenSet) == 0 {
			seedNodeTokenSetIsEmpty = true
		}
	}

	for senderKey, senderValue := range msg.MyCommNodeMap {
		if _, found := g.CommNodeMap[senderKey]; !found {
			// local nodeMap does not contain the node in sender's nodeMap
			g.CommNodeMap[senderKey] = senderValue
			for _, rnge := range senderValue.TokenSet {
				g.VirtualNodeMap[rnge] = senderValue
			}
			senderUniqueNodeCounter += 1
		} else {
			if senderKey != GetLocalNodeID() {
				if senderValue.StatusDown != g.CommNodeMap[senderKey].StatusDown {
					url := CONN_TYPE + senderValue.ContainerName + CONN_PORT + "/checkHeartbeat"
					if g.verifyNodeDown(url) == senderValue.StatusDown {
						g.CommNodeMap[senderKey] = senderValue
					} else {
						fmt.Println("Local copy is correct")
					}
				} else if len(g.CommNodeMap[senderKey].TokenSet) < len(senderValue.TokenSet) {
					g.CommNodeMap[senderKey] = senderValue
				}
			}
			commonNodeCounter += 1
		}
	}
	fmt.Println("Updated node map from inputs ", g.CommNodeMap)
	if iGotUniqueNodes := len(g.CommNodeMap) - commonNodeCounter - senderUniqueNodeCounter; iGotUniqueNodes > 0 || seedNodeTokenSetIsEmpty {
		fmt.Println(GetLocalContainerName(), "server has unique nodes!")
		updateForSender = true
		for myKey, myValue := range g.CommNodeMap {
			if _, found := msg.MyCommNodeMap[myKey]; !found {
				nodeMapForSender[myKey] = myValue
			}
			for _, seedNodeID := range seedNodesArr {
				if myKey == seedNodeID {
					nodeMapForSender[seedNodeID] = myValue
				}
			}
		}

	} else {
		updateForSender = false
		fmt.Println("No unique nodes in server!")
	}
	g.mu.Unlock()
	gossipMessage := GossipMessage{Update: updateForSender, ContainerName: GetLocalContainerName(), MyCommNodeMap: nodeMapForSender}
	return gossipMessage
}

//  General helper functions

func (g *GossipNode) GetId() int {
	id, err := strconv.Atoi(os.Getenv("NODE_ID"))
	if err != nil {
		fmt.Println(err)
	}
	return id
}

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
		return
	}
}

// func deepCopyMap(originalMap map[string]Node) map[string]Node {
// 	newMap := make(map[string]Node)
// 	for key, value := range originalMap {
// 		newMap[key] = value
// 	}
// 	return newMap
// }
