package main

import (
	"ShoppiDB/pkg/data_versioning"
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/k0kubun/pp/v3"

	http_api "ShoppiDB/pkg/http_api"
)

var node data_versioning.Node
var localDataObject data_versioning.DataObject

func main() {
	id := os.Getenv("NODE_ID")
	go http_api.StartHTTPServer()
	// ln, err := net.Listen("tcp", ":8080")
	// if err != nil {
	// 	log.Fatal("server, Listen", err)
	// }
	// go listenMessage(ln)
	// for {
	// 	time.Sleep(time.Second * 1)
	// 	sendMessage(id)
	// }
	httpClient := http_api.GetHTTPClient()
	var nodeIds []int
	nodeId, err := strconv.Atoi(getNode(id))
	if err != nil {
		fmt.Println("ISSUE WITH CLIENT NODE ID")
	}
	node = data_versioning.Node(id)
	localDataObject = data_versioning.NewDataObject("1", nil)
	go listenMessage(ln)
	for {
		data_versioning.UpdateVectorClock(node, localDataObject.Version)
		time.Sleep(time.Second * 5)
		sendMessage()
	}
}

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

func sendMessage() {
	fmt.Printf("Node %s sending message\n", node)
	target := getNode()
	time.Sleep(time.Millisecond * 5)
	con, err := net.Dial("tcp", target)
	encoder := gob.NewEncoder(con)
	_ = encoder.Encode(&localDataObject)

	// func sendMessage(id string) {
	// 	fmt.Println(id)
	// 	fmt.Println("Sending message")
	// 	target, msg := getNode(id)
	// 	time.Sleep(time.Millisecond * 1)
	// 	con, err := net.Dial("tcp", target)

	checkErr(err)
}

func getNode() string {
	switch node {
	case "1":
		return "node2:8080"
	case "2":
		return "node1:8080"
	default:
		fmt.Println("ERROR ID")
		return "null"
	}
}

func checkErr(err error) {
	if err != nil {
		fmt.Println("CONNECTION ERROR")
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
