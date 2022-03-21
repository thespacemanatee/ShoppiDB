package replication

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

type Node struct {
	Id               string
	N                int
	R                int
	W                int
	NodeStructure    [4]int
	ReplicationCheck map[string]bool
	Rbd              redis.Client
}

func (node *Node) Start() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal("server Listen", err)
	}
	go node.listenMessage(ln)
}

func (node *Node) listenMessage(ln net.Listener) {
	defer ln.Close()
	fmt.Println("Node: " + (node.Id) + " Start listening")
	fmt.Println("Node: " + (node.Id) + " Listening on :8080")
	fmt.Println("Node: " + (node.Id) + " Waiting for client...")
	for {
		// get message, output
		connection, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		fmt.Println("client connected")
		go node.processClient(connection)
	}
}

func (node *Node) processClient(connection net.Conn) {
	buffer := make([]byte, 1024)
	mLen, err := connection.Read(buffer)
	if err != nil {
		fmt.Println("Error reading:", err.Error())
	}
	data := string(buffer[:mLen])
	receivedParts := strings.Split(data, " ")
	sourceAddr := strings.TrimSpace(receivedParts[0])
	receivedMsg := strings.TrimSpace(receivedParts[1])
	if receivedMsg == "failed" {
		fmt.Println("Node: "+(node.Id)+" Received: ", data)
		fmt.Println("Node: " + (node.Id) + " FAILED")
	} else if receivedMsg == "success" {
		fmt.Println("Node: "+(node.Id)+" Received: ", data)
		fmt.Println("Node: " + (node.Id) + " SUCCESS")
		node.ReplicationCheck[sourceAddr] = true
	} else if receivedMsg == "hintedHandoff" {
		go handleHandoff()
	}	else {
		fmt.Println("Node: "+(node.Id)+" Received: ", data)
		ctx := context.TODO()
		dbErr := node.Rbd.Set(ctx, receivedMsg, receivedMsg, time.Second*5).Err()
		var msg string
		if dbErr != nil {
			msg = "failed"
		} else {
			msg = "success"
		}
		fmt.Println(msg)
		go replyMessage(sourceAddr, node.getOwnAdress()+" "+msg)
	}
	connection.Close()
}

func replyMessage(nodeAddress string, msg string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovering from panic", r)
		}
	}()
	fmt.Println("sending reply message to " + nodeAddress)
	conn, err := net.Dial("tcp", nodeAddress)
	checkErr(err)
	defer conn.Close()
	_, err = conn.Write([]byte(msg))
	checkErr(err)
}

func (node *Node) ReplicateWrites() {
	fmt.Println("Node: " + (node.Id) + " begining replication")
	node.ReplicationCheck = make(map[string]bool)
	failedConnections := 0
	// target and msg needs to change to take into account the various nodes to send
	targets, msg := node.getNode()
	time.Sleep(time.Millisecond * 1)
	for _, target := range targets {
		res := handleReplicationConnection(target, msg)
		failedConnections += res
	}
	// get the next few nodes due to failure
	id, err := strconv.Atoi(node.Id)
	if err != nil {
		fmt.Println(err)
	}
	startingPt := node.N + id
	for {
		newTargets, _ := node.getMoreNodes(failedConnections, startingPt)
		for _, target := range newTargets {
			res := handleReplicationConnection(target, msg)
			if res == 0 {
				failedConnections--
				startingPt++
				if startingPt == id {
					fmt.Println("not enough nodes available")
					break
				}
			}
		}
		if failedConnections == 0 {
			break
		}
	}

	time.Sleep(time.Second * 5)
	fmt.Println("THIS IS THE RESULTS")
	fmt.Println(node.ReplicationCheck)
	res := node.checkWriteQuorum()
	fmt.Println(res)
}

func handleReplicationConnection(target string, msg string) (failedConn int) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("panic occured", r)
			failedConn = 1
		}
	}()
	fmt.Println("This is the target: " + target)
	con, err := net.Dial("tcp", target)
	checkErr(err)
	defer con.Close() //Requires to catch the null error when fail to connect before writing
	_, err = con.Write([]byte(msg))
	checkErr(err)
	return 0
}

func checkErr(err error) {

	if err != nil {
		fmt.Println("CONNECTION ERROR")
		fmt.Println(err)
	}
}

func (node *Node) getMoreNodes(numNodes int, startingPt int) ([]string, string) {
	nodesTosend := make([]string, numNodes)
	count := 0
	passedLastNode := false
	for i := startingPt; i < startingPt+numNodes; i++ {
		switch strconv.Itoa(i % (len(node.NodeStructure) + 1)) {
		default:
			fmt.Println("ERROR ID")
		case "0":
			nodesTosend[count] = "node1:8080"
			passedLastNode = true
		case "1":
			if !passedLastNode {
				nodesTosend[count] = "node1:8080"
			} else {
				nodesTosend[count] = "node2:8080"
			}
		case "2":
			if !passedLastNode {
				nodesTosend[count] = "node2:8080"
			} else {
				nodesTosend[count] = "node3:8080"
			}
		case "3":
			if !passedLastNode {
				nodesTosend[count] = "node3:8080"
			} else {
				nodesTosend[count] = "node4:8080"
			}
		case "4":
			if !passedLastNode {
				nodesTosend[count] = "node4:8080"
			} else {
				nodesTosend[count] = "node1:8080"
			}
		}
		count++
	}
	return nodesTosend, node.getOwnAdress() + " " + "hintedHandoff"
}

func (node *Node) getNode() ([]string, string) {
	// depending on n value
	nodesTosend := make([]string, node.N-1) // -1 because one copy stored locally
	passedLastNode := false
	for i := 1; i < node.N; i++ {
		nodeID, err := strconv.Atoi(node.Id)
		// supposed to modulo the total number of nodes but ring structure not available
		fmt.Println("This is nodeID + i % n" + strconv.Itoa((nodeID+i)%(len(node.NodeStructure)+1)))
		if err != nil {
			fmt.Println(err)
		} else {
			switch strconv.Itoa((nodeID + i) % (len(node.NodeStructure) + 1)) {
			default:
				fmt.Println("ERROR ID")
				//nodesTosend[i-1] = "null"
			case "0":
				nodesTosend[i-1] = "node1:8080"
				passedLastNode = true
			case "1":
				if !passedLastNode {
					nodesTosend[i-1] = "node1:8080"
				} else {
					nodesTosend[i-1] = "node2:8080"
				}
			case "2":
				if !passedLastNode {
					nodesTosend[i-1] = "node2:8080"
				} else {
					nodesTosend[i-1] = "node3:8080"
				}
			case "3":
				if !passedLastNode {
					nodesTosend[i-1] = "node3:8080"
				} else {
					nodesTosend[i-1] = "node4:8080"
				}
			case "4":
				if !passedLastNode {
					nodesTosend[i-1] = "node4:8080"
				} else {
					nodesTosend[i-1] = "node1:8080"
				}
			}
		}
	}
	fmt.Println("nodes to send ")
	fmt.Println(nodesTosend)
	return nodesTosend, node.getOwnAdress() + " " + "replication"
}

func (node *Node) checkWriteQuorum() bool {
	values := []bool{}
	for _, v := range node.ReplicationCheck {
		if v {
			values = append(values, v)
		}
	}
	fmt.Println(len(values))
	if len(values) < node.W {
		fmt.Println("failed write quorum")
		return false
	} else {
		fmt.Println("passed write quorum")
		return true
	}
}

func (node *Node) getOwnAdress() string {
	switch node.Id {
	default:
		return ""
	case "1":
		return "node1:8080"
	case "2":
		return "node2:8080"
	case "3":
		return "node3:8080"
	case "4":
		return "node4:8080"
	}
}
