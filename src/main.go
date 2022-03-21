package main

import (
	"ShoppiDB/src/replication"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

// type Message struct {
// 	senderID string `json: "senderID`
// 	messasge string `json: "message`
// }

func main() {
	client := redis.NewClient(&redis.Options{})
	//case where replication goes around the ring once
	//node 2 with replicas in 3, 4 and 1
	// nodeStructure := [4]int{1, 2, 3, 4}
	// node := Node{os.Getenv("NODE_ID"), 4, 1, 1, nodeStructure,make(map[string]bool), *client}
	// node.Start()
	// for {
	// 	time.Sleep(time.Second * 1)
	// 	if node.id == strconv.Itoa(2) {
	// 		node.ReplicateWrites()
	// 		time.Sleep(time.Second * 10)
	// 	}
	// }
	// fmt.Println("End of Program")

	// case where all nodes are sending replications
	// nodeStructure := [4]int{1, 2, 3, 4}
	// node := Node{os.Getenv("NODE_ID"), 4, 1, 1, nodeStructure, make(map[string]bool),*client}
	// node.Start()
	// for {
	// 	time.Sleep(time.Second * 1)
	// 	node.ReplicateWrites()
	// 	time.Sleep(time.Second * 10)
	// }
	// fmt.Println("End of Program")

	// case where 1 node is sending replication and n=2
	// nodeStructure := [4]int{1, 2, 3, 4}
	// node := Node{os.Getenv("NODE_ID"), 2, 1, 1, nodeStructure, make(map[string]bool), *client}
	// node.Start()
	// for {
	// 	time.Sleep(time.Second * 1)
	// 	if node.id == strconv.Itoa(1) {
	// 		node.ReplicateWrites()
	// 		time.Sleep(time.Second * 10)
	// 	}
	// }
	// fmt.Println("End of Program")

	//case where node 2 is sleeping N=3 and node 1 sends, expects node 3 and 4 to receive
	// nodeStructure := [4]int{1, 2, 3, 4}
	// node := replication.Node{Id: os.Getenv("NODE_ID"), N: 3, R: 1, W: 4, NodeStructure: nodeStructure, ReplicationCheck: make(map[string]bool), Rbd: *client}
	// if node.Id != strconv.Itoa(2) {
	// 	node.Start()
	// }
	// sleep := true
	// for {
	// 	if node.Id == strconv.Itoa(2) && sleep {
	// 		time.Sleep(time.Second * 30)
	// 		sleep = false
	// 		fmt.Println("NODE WOKE UP")
	// 		node.Start()

	// 	}
	// 	if node.Id == strconv.Itoa(1) {
	// 		time.Sleep(time.Millisecond * 100)
	// 		node.ReplicateWrites()
	// 		time.Sleep(time.Second * 10)
	// 	}
	// }
	// fmt.Println("End of Program")

	//case where node 2 and 3 is sleeping n=2 and node 1 sends, N=2, expects node 4 to receive
	nodeStructure := [4]int{1, 2, 3, 4}
	node := replication.Node{Id: os.Getenv("NODE_ID"), N: 3, R: 1, W: 4, NodeStructure: nodeStructure, ReplicationCheck: make(map[string]bool), Rbd: *client}
	if node.Id != strconv.Itoa(2) {
		node.Start()
	}
	sleep := true
	for {
		if (node.Id == strconv.Itoa(2) || node.Id == strconv.Itoa(3)) && sleep {
			time.Sleep(time.Second * 30)
			sleep = false
			fmt.Println("NODE WOKE UP")
			node.Start()

		}
		if node.Id == strconv.Itoa(1) {
			time.Sleep(time.Millisecond * 100)
			node.ReplicateWrites()
			time.Sleep(time.Second * 10)
		}
	}
	fmt.Println("End of Program")

}
