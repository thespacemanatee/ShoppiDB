package main

import (
	"ShoppiDB/pkg/data_versioning"
	nodePkg "ShoppiDB/pkg/node"
	"encoding/gob"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/k0kubun/pp/v3"
)

var node data_versioning.Node
var localDataObject data_versioning.DataObject

func main() {
	// done := make(chan struct{})
	// localNode := gossip.Node{Membership: gossip.GetMembership(), ContainerName: gossip.GetLocalContainerName()}
	// toCommunicate := gossip.Gossip{NodeMap: make(map[string]gossip.Node)}

	// //adding localNode into node map
	// toCommunicate.NodeMap[gossip.GetLocalNodeID()] = localNode

	// go toCommunicate.ServerStart()
	// go toCommunicate.ClientStart()
	// time.Sleep(time.Minute * 5)

	id := os.Getenv("NODE_ID")
	node := nodePkg.Node{}
	go node.StartHTTPServer()
	httpClient := nodePkg.GetHTTPClient()
	var oppId int
	switch id {
	case "1":
		oppId = 2
	case "2":
		oppId = 1
	default:
		oppId = 0
	}
	nodeDNS, err := getNodeDNS(oppId)
	checkErr(err)
	time.Sleep(time.Second * 5) //Buffer time to start HTTPSERVER
	for {
		value := generateNonce()
		node.DbPUT(nodeDNS, httpClient, value)
		time.Sleep(1 * time.Second)
		node.DbGET(nodeDNS, httpClient)
		// node.BasicHTTPGET(nodeDNS, httpClient)

	}
}

func generateNonce() string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, 8)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
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
