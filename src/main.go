package main

import (
	consistent_hashing "ShoppiDB/pkg/consistent_hashing"
	replication "ShoppiDB/pkg/data_replication"
	"ShoppiDB/pkg/data_versioning"
	gossip "ShoppiDB/pkg/gossip"
	nodePkg "ShoppiDB/pkg/node"
	"ShoppiDB/pkg/redisDB"
	"container/heap"
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/k0kubun/pp/v3"
)

var localDataObject data_versioning.DataObject

func main() {
	id := os.Getenv("NODE_ID")
	idInt, err := strconv.Atoi(id)
	if err != nil {
		fmt.Println(err)
	}
	tokenSet := nodePkg.GenTokenSet()
	gossipNode := gossip.GossipNode{Id: idInt, ContainerName: gossip.GetLocalContainerName(), Membership: gossip.GetMembership(), TokenSet: tokenSet}
	localCommNodeMap := make(map[string]gossip.GossipNode)
	localCommNodeMap[gossip.GetLocalNodeID()] = gossipNode
	httpClient := nodePkg.GetHTTPClient(300 * time.Millisecond)
	longerTOClient := nodePkg.GetHTTPClient(1 * time.Second)
	localVirtualNodeMap := make(map[[2]int]gossip.GossipNode)
	for _, rnge := range tokenSet {
		localVirtualNodeMap[rnge] = gossipNode
	}
	priorityQueue := make(replication.PriorityQueue, 0)
	heap.Init(&priorityQueue)
	replicator := replication.Replicator{Id: id, N: 2, W: 1, R: 2, HttpClient: httpClient, LongerTOClient: longerTOClient, Rdb: *redisDB.GetDBClient(), Queue: priorityQueue}
	localNode := nodePkg.Node{Replicator: &replicator, Membership: gossip.GetMembership(), ContainerName: gossip.GetLocalContainerName(), TokenSet: tokenSet, Gossiper: gossip.Gossip{CommNodeMap: localCommNodeMap, VirtualNodeMap: localVirtualNodeMap}}
	fmt.Println(gossip.GetLocalContainerName(), "STARTING")
	key := "hello world"
	hashKey := consistent_hashing.GetMD5Hash(key)
	go localNode.StartHTTPServer()
	time.Sleep(time.Second * 10)
	go localNode.Gossiper.Start()
	time.Sleep(time.Second * 20)
	fmt.Println(hashKey)
	for {

	}
}

//Example Code for socket
func listenMessage(ln net.Listener) {
	fmt.Println("Start listening")
	// accept connection
	defer ln.Close()
	fmt.Println("Listening on :8080")
	fmt.Println("Waiting for client...")
	for {
		// get message, output
		connection, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		go processClient(connection)
	}
}

//Example code for sending message through socket
func sendMessage(id int) {
	target := "Tobechange"
	time.Sleep(time.Millisecond * 5)
	con, err := net.Dial("tcp", target)
	encoder := gob.NewEncoder(con)
	_ = encoder.Encode(&localDataObject)

	checkErr(err)
}

//Get the respective DN of the nodes
func getNodeDNS(i interface{}) (string, error) {
	switch o := i.(type) {
	case string:
		return "node" + o, nil
	case int:
		return "node" + strconv.Itoa(o), nil
	default:
		return "", errors.New("Invalid Format")
	}
}

//Basic error print
func checkErr(err error) {
	if err != nil {
		fmt.Println("ERROR")
		fmt.Println(err)
	}
}

func processClient(connection net.Conn) {
	dec := gob.NewDecoder(connection)
	dataObject := &data_versioning.DataObject{}
	err := dec.Decode(dataObject)
	objects := []data_versioning.DataObject{localDataObject, *dataObject}
	newObjects := data_versioning.GetResponseDataObjects(objects)
	pp.Printf("Response objects: %+v\n", newObjects)
	if err != nil {
		fmt.Println("Error reading:", err.Error())
	}
	pp.Printf("Received: %+v\n", dataObject)
	_ = connection.Close()
}
