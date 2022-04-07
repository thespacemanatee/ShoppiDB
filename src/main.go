package main

import (
	"ShoppiDB/pkg/consistent_hashing"
	"ShoppiDB/pkg/data_versioning"
	gossip "ShoppiDB/pkg/gossip"
	nodePkg "ShoppiDB/pkg/node"
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/k0kubun/pp/v3"
)

var node data_versioning.Node
var localDataObject data_versioning.DataObject

func main() {
	gossipNode := gossip.GossipNode{ContainerName: gossip.GetLocalContainerName(), Membership: gossip.GetMembership()}
	localCommNodeMap := make(map[string]gossip.GossipNode)
	localCommNodeMap[gossip.GetLocalNodeID()] = gossipNode
	httpClient := nodePkg.GetHTTPClient()
	localNode := nodePkg.Node{Membership: gossip.GetMembership(), ContainerName: gossip.GetLocalContainerName(), Gossiper: gossip.Gossip{CommNodeMap: localCommNodeMap, HttpClient: httpClient}}

	fmt.Println(gossip.GetLocalContainerName(), "STARTING")

	go localNode.StartHTTPServer()
	time.Sleep(time.Second * 10)
	go localNode.Gossiper.Start()
	time.Sleep(time.Minute * 5)
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
	fmt.Printf("Node %s sending message\n", node)
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
