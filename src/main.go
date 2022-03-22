package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	byzantine "ShoppiDB/pkg/byzantine"
	http_api "ShoppiDB/pkg/http_api"
)

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
	nodeIds = append(nodeIds, nodeId)
	clientId, err := strconv.Atoi(id)
	if err != nil {
		fmt.Println("ISSUE WITH ENV NODE ID")
	}
	for {
		time.Sleep(time.Second * 5)
		byzantine.SendByzantineInitiateRead(httpClient, clientId, nodeIds, id)
	}
}

// func listenMessage(ln net.Listener) {
// 	fmt.Println("Start listening")
// 	// accept connection
// 	defer ln.Close()
// 	fmt.Println("Listening on :8080")
// 	fmt.Println("Waiting for client...")
// 	for {
// 		// get message, output
// 		connection, err := ln.Accept()
// 		if err != nil {
// 			fmt.Println("Error accepting: ", err.Error())
// 			os.Exit(1)
// 		}
// 		fmt.Println("client connected")
// 		go processClient(connection)
// 	}
// }

// func sendMessage(id string) {
// 	fmt.Println(id)
// 	fmt.Println("Sending message")
// 	target, msg := getNode(id)
// 	time.Sleep(time.Millisecond * 1)
// 	con, err := net.Dial("tcp", target)

// 	defer con.Close()

// 	checkErr(err)

// 	_, err = con.Write([]byte(msg))

// 	checkErr(err)
// }

func getNode(id string) string {
	switch id {
	default:
		fmt.Println("ERROR ID")
		return "1"
	case "1":
		return "2"
	case "2":
		return "1"
	}
}

// func processClient(connection net.Conn) {
// 	buffer := make([]byte, 1024)
// 	mLen, err := connection.Read(buffer)
// 	if err != nil {
// 		fmt.Println("Error reading:", err.Error())
// 	}
// 	fmt.Println("Received: ", string(buffer[:mLen]))
// 	connection.Close()
// }
