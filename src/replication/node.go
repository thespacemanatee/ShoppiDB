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
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovering from panic", r)
			connection.Close()
		}
	}()
	buffer := make([]byte, 1024)
	mLen, err := connection.Read(buffer)
	if err != nil {
		fmt.Println("Error reading:", err.Error())
	}
	data := string(buffer[:mLen])
	splitMsg := strings.Split(data, " ")
	sourceAddr := strings.TrimSpace(splitMsg[0])
	receivedMsg := strings.TrimSpace(splitMsg[1])
	if receivedMsg == "failed" {
		fmt.Println("Node: "+(node.Id)+" Received: ", data)
		fmt.Println("Node: " + (node.Id) + " FAILED")
	} else if receivedMsg == "success" {
		fmt.Println("Node: "+(node.Id)+" Received: ", data)
		fmt.Println("Node: " + (node.Id) + " SUCCESS")
		node.ReplicationCheck[sourceAddr] = true
	} else if strings.Contains(receivedMsg, "hintedHandoff") {
		msg := strings.Split(receivedMsg, "hintedHandoff")
		fmt.Println("hinted handoff data " + msg[1])
		go node.handleHandoff(node.getOwnAdress(), msg[1], node.getOwnAdress()+" handoff data")
	} else {
		fmt.Println("Node: "+(node.Id)+" Received: ", data)
		go node.handleResponse(sourceAddr, receivedMsg)
	}
	connection.Close()
}

func (node *Node) ReplicateWrites() {
	fmt.Println("Node: " + (node.Id) + " begining replication")
	id, err := strconv.Atoi(node.Id)
	if err != nil {
		fmt.Println(err)
	}
	node.ReplicationCheck = make(map[string]bool)
	failedConnections := 0
	failedTargets := []string{}
	targets, msg := node.getNodes(node.N-1, id+1, true)
	for _, target := range targets {
		res, t := handleReplicationConnection(target, msg)
		if res == 1 {
			failedTargets = append(failedTargets, t)
		}
		failedConnections += res
	}
	// handle failed connections by trying more nodes in the preference list
	startingPt := node.N + id
	count := 0
	for {
		newTargets, newMsg := node.getNodes(failedConnections, startingPt, false)
		for _, target := range newTargets {
			res, _ := handleReplicationConnection(target, newMsg+failedTargets[count])
			if res == 0 {
				count++
				failedConnections--
				if startingPt == id {
					fmt.Println("not enough nodes available")
					break
				}
			} else {
				startingPt++
			}
		}
		if failedConnections == 0 {
			break
		}
	}
	time.Sleep(time.Second * 3)
	fmt.Println("THIS IS THE RESULTS")
	fmt.Println(node.ReplicationCheck)
	res := node.checkWriteQuorum()
	fmt.Println(res)
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

func handleReplicationConnection(target string, msg string) (failedConn int, failedTarget string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("panic occured", r)
			failedConn = 1
			failedTarget = target
		}
	}()
	fmt.Println("This is the target: " + target)
	con, err := net.Dial("tcp", target)
	checkErr(err)
	defer con.Close() //Requires to catch the null error when fail to connect before writing
	_, err = con.Write([]byte(msg))
	checkErr(err)
	return 0, "passed"
}

func checkErr(err error) {
	if err != nil {
		fmt.Println("CONNECTION ERROR")
		fmt.Println(err)
	}
}

func (node *Node) getNodes(numNodes int, startingPt int, isReplication bool) ([]string, string) {
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
	if isReplication {
		return nodesTosend, node.getOwnAdress() + " " + "replication"
	}
	return nodesTosend, node.getOwnAdress() + " " + "hintedHandoff"
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

func (node *Node) handleHandoff(sourceAddr string, intendedRecipient, receivedMsg string) {
	ctx := context.TODO()
	dbErr := node.Rbd.Set(ctx, receivedMsg, receivedMsg, 0).Err()
	var msg string
	if dbErr != nil {
		msg = "failed"
	} else {
		msg = "success"
	}
	go replyMessage(sourceAddr, node.getOwnAdress()+" "+msg)
	for {
		res, _ := handleReplicationConnection(intendedRecipient, receivedMsg)
		if res == 0 {
			break
		}
		time.Sleep(time.Second * 2)
	}

}

func (node *Node) handleResponse(sourceAddr string, receivedMsg string) {
	ctx := context.TODO()
	dbErr := node.Rbd.Set(ctx, receivedMsg, receivedMsg, time.Second*5).Err()
	var msg string
	if dbErr != nil {
		msg = "failed"
	} else {
		msg = "success"
	}
	go replyMessage(sourceAddr, node.getOwnAdress()+" "+msg)
}
