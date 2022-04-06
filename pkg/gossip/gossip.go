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
	nodePkg "ShoppiDB/pkg/node"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	CONN_PORT = ":8080"
	CONN_TYPE = "tcp"
)

// Global gossip variable declaration
var gossip Gossip

type Node struct {
	Membership    bool
	ContainerName string
	// nodeID
	// tokenSet
	// timeOfIssue int
}

type Gossip struct {
	mu             sync.Mutex
	CommNodeMap    map[string]nodePkg.Node
	VirtualNodeMap map[[2]int]nodePkg.Node
}

type Message struct {
	Msg string
}

/*
CLIENT
*/

func (g *Gossip) ClientStart() {
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
	ticker := time.NewTicker(10 * time.Second)
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
				con, err := net.Dial(CONN_TYPE, seedNode.ContainerName+CONN_PORT)
				checkErr(err)
				g.sendMyNodeMap(con)
				g.waitForResponse(con)
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
		con, err := net.Dial(CONN_TYPE, randNode.ContainerName+CONN_PORT)
		checkErr(err)
		g.sendMyNodeMap(con)
		g.waitForResponse(con)
	}
}

// Helper functions for gossip.clientStart
func (g *Gossip) populate(seedNodesArray []string) map[string]Node {
	g.mu.Lock()
	seedNodesMap := make(map[string]Node)
	for _, str := range seedNodesArray {
		node := Node{Membership: true, ContainerName: nodeidToContainerName(str)}
		g.NodeMap[str] = node
		// Populating seedNode map as well
		seedNodesMap[str] = node
	}
	g.mu.Unlock()
	return seedNodesMap
}

func (g *Gossip) getRandNode() Node {
	g.mu.Lock()
	randomInt := rand.Intn(len(g.NodeMap))
	fmt.Println("string(randomInt):", string(randomInt))
	randNode := g.NodeMap[string(randomInt)]
	g.mu.Unlock()
	return randNode
}

func (g *Gossip) waitForResponse(con net.Conn) {
	response := make([]byte, 1024)
	fmt.Println("Waiting for server's response")
	msgLen, errResp := con.Read(response)
	checkErr(errResp)
	reply := string(response[:msgLen])
	fmt.Println("Server's response is:", reply)
	if reply == "no" {
		con.Close()
	} else {
		g.recvNodes(con)
	}
}

func (g *Gossip) recvNodes(con net.Conn) {
	g.mu.Lock()
	dec := gob.NewDecoder(con)
	var incNodeMap map[string]Node
	err := dec.Decode(&incNodeMap)
	checkErr(err)
	fmt.Println(GetLocalContainerName()+" has received", incNodeMap)

	for key, value := range incNodeMap {
		g.NodeMap[key] = value
	}
	g.mu.Unlock()
	con.Close()
}

func (g *Gossip) sendMyNodeMap(con net.Conn) {
	// localNode := g.nodeMap[getLocalNodeID()]
	myNodeMap := deepCopyMap(g.NodeMap)
	// myNodeMap := g.nodeMap //pass by value
	enc := gob.NewEncoder(con)
	errEnc := enc.Encode(myNodeMap)
	checkErr(errEnc)
	fmt.Println(GetLocalContainerName()+" has sent", myNodeMap)
}

/*
SERVER
*/

func GossipHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(GetLocalContainerName(), "HAS RECEIVED MESSAGE!")
	w.Header().Set("Content-Type", "application/json")
	if r.Body == nil {
		http.Error(w, "Please send a request body", 400)
		return
	}
	var message Message
	err := json.NewDecoder(r.Body).Decode(&message)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	// Business logic
	fmt.Println(message.Msg)
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(message) //Writing the message back
}

// func (g *Gossip) ServerStart() {
// 	fmt.Println("Starting server...")
// 	dataStream, err := net.Listen(CONN_TYPE, CONN_PORT)
// 	checkErr(err)
// 	defer dataStream.Close()
// 	for {
// 		con, err := dataStream.Accept()
// 		checkErr(err)
// 		go g.listenMsg(con)
// 	}
// }

//Helper functions for gossip.serverStart
func (g *Gossip) listenMsg(con net.Conn) {
	dec := gob.NewDecoder(con)
	var senderNodeMap map[string]Node
	err := dec.Decode(&senderNodeMap)
	checkErr(err)
	fmt.Println(GetLocalContainerName()+" has received", senderNodeMap)
	updateForSender, returningNodeMap := g.compareAndUpdate(senderNodeMap)
	sendMsg(con, updateForSender)
	if updateForSender == "yes" {
		// Need to give some time for message to be fully sent through the tcp conn
		time.Sleep(time.Millisecond * 50)
		sendUpdateNodeMap(con, returningNodeMap)
	}
}

func (g *Gossip) compareAndUpdate(senderNodeMap map[string]Node) (string, map[string]Node) {
	g.mu.Lock()
	fmt.Println("senderNodeMap:", senderNodeMap)
	fmt.Println("myNodeMap:", g.NodeMap)
	updateForSender := "no"
	nodeMapForSender := make(map[string]Node)
	commonNodeCounter := 0
	senderUniqueNodeCounter := 0

	for senderKey, senderValue := range senderNodeMap {
		if _, found := g.NodeMap[senderKey]; !found {
			// local nodeMap does not contain the node in sender's nodeMap
			g.NodeMap[senderKey] = senderValue
			senderUniqueNodeCounter += 1
		} else {
			commonNodeCounter += 1
		}
	}

	if iGotUniqueNodes := len(g.NodeMap) - commonNodeCounter - senderUniqueNodeCounter; iGotUniqueNodes > 0 {
		fmt.Println("Server has unique nodes!")
		updateForSender = "yes"
		for myKey, myValue := range g.NodeMap {
			if _, found := senderNodeMap[myKey]; !found {
				nodeMapForSender[myKey] = myValue
			}
		}
	}
	g.mu.Unlock()
	fmt.Println("No unique nodes in server!")
	return updateForSender, nodeMapForSender
}

func sendMsg(con net.Conn, msg string) {
	// responseToSender := make([]byte, 1024)
	fmt.Println("Writing response to sender")
	_, err := con.Write([]byte(msg))
	checkErr(err)
}

func sendUpdateNodeMap(con net.Conn, nodeMap map[string]Node) {
	enc := gob.NewEncoder(con)
	err := enc.Encode(nodeMap)
	checkErr(err)
	fmt.Println("Server has sent update node map to client!")
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

func deepCopyMap(originalMap map[string]Node) map[string]Node {
	newMap := make(map[string]Node)
	for key, value := range originalMap {
		newMap[key] = value
	}
	return newMap
}

//HTTP
// func startHTTPServer() {
// 	fmt.Println("Starting HTTP Server for gossip")
// 	router := mux.NewRouter().StrictSlash(true)
// 	router.HandleFunc("/", defaultHandler).Methods("GET")
// 	router.HandleFunc("/byzantine", byzantineHandler).Methods("POST")
// 	router.HandleFunc("/gossip", gossip.GossipHandler).Methods("POST")
// 	log.Fatal(http.ListenAndServe(":8080", router))
// }
