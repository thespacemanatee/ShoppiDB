package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

func main() {
	id := os.Getenv("NODE_ID")
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal("server, Listen", err)
	}
	go listenMessage(ln)
	for {
		time.Sleep(time.Second * 1)
		sendMessage(id)
	}
	fmt.Println("End of Program")
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
		fmt.Println("client connected")
		go processClient(connection)
	}
}

func sendMessage(id string) {
	fmt.Println(id)
	fmt.Println("Sending message")
	target, msg := getNode(id)
	time.Sleep(time.Millisecond * 1)
	con, err := net.Dial("tcp", target)

	defer con.Close() //Requires to catch the null error when fail to connect before writing

	checkErr(err)

	_, err = con.Write([]byte(msg))

	checkErr(err)
}

func getNode(id string) (string, string) {
	switch id {
	default:
		fmt.Println("ERROR ID")
		return "null", "null"
	case "1":
		return "node2:8080", "From node 1"
	case "2":
		return "node1:8080", "From node 2"
	}
}

func checkErr(err error) {

	if err != nil {
		fmt.Println("CONNECTION ERROR")
		fmt.Println(err)
	}
}

func processClient(connection net.Conn) {
	buffer := make([]byte, 1024)
	mLen, err := connection.Read(buffer)
	if err != nil {
		fmt.Println("Error reading:", err.Error())
	}
	fmt.Println("Received: ", string(buffer[:mLen]))
	connection.Close()
}
