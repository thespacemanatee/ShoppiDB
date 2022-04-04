package replication

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

type Replicator struct {
	Id            string
	N             int
	R             int
	W             int
	NodeStructure [4]int
	WriteCheck    map[int]bool
	ReadCheck     map[int]bool
	Rbd           redis.Client
	HttpClient    *http.Client
	mu            sync.Mutex
}

type ReplicationMessage struct {
	SenderId            int    `json:"senderid"`
	Dest                int    `json:"receiverid"`
	IntendedRecipientId int    `json:"intendedreceipientid"`
	Data                string `json:"data"`
	MessageCode         int    `json:"messagecode"`
	// 0 - failed write
	// 1 - successful write
	// 2 - replication data
	// 3 - hinted handoff
	// 4 -failed read
	// 5 - successful read
	// 6 - key data
}

func (r Replicator) SendReplicationMessage(httpClient *http.Client, msg ReplicationMessage) (failedConn int, failedTarget int) {
	// used to indicate whether connection failed
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("panic occured", r)
			failedConn = 1
			failedTarget = msg.Dest
		}
	}()

	fmt.Println("Node: " + strconv.Itoa(msg.SenderId) + " Sending message type: " + strconv.Itoa(msg.MessageCode) + " to replica node: " + strconv.Itoa(msg.Dest))
	target := "http://node" + strconv.Itoa(msg.Dest) + ":8080/replication"
	replicationMessageJson, err := json.Marshal(msg)
	checkErr(err)
	req, err := http.NewRequest(http.MethodPost, target, bytes.NewBuffer(replicationMessageJson))
	checkErr(err)
	req.Header.Set("Content-Type", "application/json")
	fmt.Println(req)
	resp, err := httpClient.Do(req)
	checkErr(err)
	defer resp.Body.Close()
	var receivedMessage string
	json.NewDecoder(resp.Body).Decode(&receivedMessage)
	fmt.Println(receivedMessage)
	return 0, 0
}

// starts replication processs
// needs to be updated to use node structure from gossip
func (r *Replicator) ReplicateWrites(data string) {
	fmt.Println("Node: " + (r.Id) + " begining replication")
	id, err := strconv.Atoi(r.Id)
	if err != nil {
		fmt.Println(err)
	}
	r.WriteCheck = make(map[int]bool)
	failedConnections := 0
	failedTargets := []int{}
	targets := r.getNodes(r.N-1, id+1)
	for _, target := range targets {
		msg := ReplicationMessage{SenderId: id, Dest: target, Data: data, MessageCode: 2}
		res, t := r.SendReplicationMessage(r.HttpClient, msg)
		if res == 1 {
			fmt.Println("Message to " + strconv.Itoa(t) + " failed")
			failedTargets = append(failedTargets, t)
		}
		failedConnections += res
	}
	startingPt := r.N + id
	count := 0
	// sents to nodes next in priority list
	for {
		newTargets := r.getNodes(failedConnections, startingPt)
		for _, target := range newTargets {
			handoffMsg := ReplicationMessage{SenderId: id, Dest: target, IntendedRecipientId: failedTargets[count], Data: data, MessageCode: 3}
			res, _ := r.SendReplicationMessage(r.HttpClient, handoffMsg)
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
			if failedConnections == 0 {
				break
			}
		}
	}
	time.Sleep(time.Second * 1)
	fmt.Println("Results for write replication")
	fmt.Println(r.WriteCheck)
	res := r.checkWriteQuorum()
	fmt.Println(res)
}

// starts replication processs
// needs to be updated to use node structure from gossip
func (r *Replicator) ReplicateReads(key string) {
	fmt.Println("Node: " + (r.Id) + " begining replication")
	id, err := strconv.Atoi(r.Id)
	if err != nil {
		fmt.Println(err)
	}
	r.ReadCheck = make(map[int]bool)
	failedConnections := 0
	failedTargets := []int{}
	targets := r.getNodes(r.N-1, id+1)
	for _, target := range targets {
		msg := ReplicationMessage{SenderId: id, Dest: target, Data: key, MessageCode: 2}
		res, t := r.SendReplicationMessage(r.HttpClient, msg)
		if res == 1 {
			fmt.Println("Message to " + strconv.Itoa(t) + " failed")
			failedTargets = append(failedTargets, t)
		}
		failedConnections += res
	}
	startingPt := r.N + id
	count := 0
	// sents to nodes next in priority list
	for {
		newTargets := r.getNodes(failedConnections, startingPt)
		for _, target := range newTargets {
			handoffMsg := ReplicationMessage{SenderId: id, Dest: target, IntendedRecipientId: failedTargets[count], Data: key, MessageCode: 3}
			res, _ := r.SendReplicationMessage(r.HttpClient, handoffMsg)
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
			if failedConnections == 0 {
				break
			}
		}
	}
	time.Sleep(time.Second * 1)
	fmt.Println("Results for write replication")
	fmt.Println(r.WriteCheck)
	res := r.checkReadQuorum()
	fmt.Println(res)
}

func checkErr(err error) {
	if err != nil {
		fmt.Println("CONNECTION ERROR")
		fmt.Println(err)
	}
}

//function returns the node ids to send for replication
// needs to be changed to get node ids from gossip
func (r *Replicator) getNodes(numNodes int, startingPt int) []int {
	nodesTosend := make([]int, numNodes)
	count := 0
	passedLastNode := false
	for i := startingPt; i < startingPt+numNodes; i++ {
		switch strconv.Itoa(i % (len(r.NodeStructure) + 1)) {
		default:
			fmt.Println("ERROR ID")
		case "0":
			nodesTosend[count] = 1
			passedLastNode = true
		case "1":
			if !passedLastNode {
				nodesTosend[count] = 1
			} else {
				nodesTosend[count] = 2
			}
		case "2":
			if !passedLastNode {
				nodesTosend[count] = 2
			} else {
				nodesTosend[count] = 3
			}
		case "3":
			if !passedLastNode {
				nodesTosend[count] = 3
			} else {
				nodesTosend[count] = 4
			}
		case "4":
			if !passedLastNode {
				nodesTosend[count] = 4
			} else {
				nodesTosend[count] = 1
			}
		}
		count++
	}
	return nodesTosend
}

func (r *Replicator) checkWriteQuorum() bool {
	fmt.Println("in check write quorum")
	fmt.Println(r.WriteCheck)
	values := []bool{}
	r.mu.Lock()
	for _, v := range r.WriteCheck {
		if v {
			values = append(values, v)
		}
	}
	r.mu.Unlock()
	fmt.Println(len(values))
	if len(values) < r.W {
		fmt.Println("failed write quorum")
		return false
	} else {
		fmt.Println("passed write quorum")
		return true
	}
}

func (r *Replicator) checkReadQuorum() bool {
	fmt.Println("in check write quorum")
	fmt.Println(r.ReadCheck)
	values := []bool{}
	r.mu.Lock()
	for _, v := range r.ReadCheck {
		if v {
			values = append(values, v)
		}
	}
	r.mu.Unlock()
	fmt.Println(len(values))
	if len(values) < r.R {
		fmt.Println("failed write quorum")
		return false
	} else {
		fmt.Println("passed write quorum")
		return true
	}
}

// stores handoff data in redis and tries to send message
func (r *Replicator) HandleHandoff(msg ReplicationMessage) {
	// needs to add portion for writing to redis
	// currently always assume able to write
	replyMsg := ReplicationMessage{SenderId: getOwnId(), Dest: msg.SenderId, MessageCode: 1}
	// send message to indicate handoff message received
	go r.SendReplicationMessage(r.HttpClient, replyMsg)
	msgToIntended := ReplicationMessage{SenderId: getOwnId(), Dest: msg.IntendedRecipientId, Data: msg.Data, MessageCode: 2}
	for {
		res, _ := r.SendReplicationMessage(r.HttpClient, msgToIntended)
		if res == 0 {
			break
		}
		// tries to send again in 1 second
		time.Sleep(time.Second * 1)
	}
	// delete handoff data from node
	fmt.Println("successfully sent handoff data")
}

func (r *Replicator) HandleWriteResponse(receivedMsg ReplicationMessage) {
	// need add part for writes to redis
	// successful write to redis
	msg := ReplicationMessage{SenderId: getOwnId(), Dest: receivedMsg.SenderId, Data: "string", MessageCode: 1}
	go r.SendReplicationMessage(r.HttpClient, msg)
}

func (r *Replicator) HandleReadResponse(receivedMsg ReplicationMessage) {
	// need add part for writes to redis
	// successful write to redis
	msg := ReplicationMessage{SenderId: getOwnId(), Dest: receivedMsg.SenderId, Data: "string", MessageCode: 5}
	go r.SendReplicationMessage(r.HttpClient, msg)
}

func getOwnId() int {
	id := os.Getenv("NODE_ID")
	idNum, err := strconv.Atoi(id)
	checkErr(err)
	return idNum
}

// used for successful write responses
func (r *Replicator) AddSuccessfulWrite(id int) {
	r.mu.Lock()
	r.WriteCheck[id] = true
	r.mu.Unlock()
	fmt.Println(r.WriteCheck)
}

func (r *Replicator) AddSuccessfulRead(id int) {
	r.mu.Lock()
	r.ReadCheck[id] = true
	r.mu.Unlock()
	fmt.Println(r.ReadCheck)
}
