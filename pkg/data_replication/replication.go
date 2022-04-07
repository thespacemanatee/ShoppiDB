package replication

import (
	"ShoppiDB/pkg/redisDB"
	"bytes"
	"context"
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
	HandoffCheck  map[int]bool
	ReadCheck     map[int]string
	Rdb           redis.Client
	HttpClient    *http.Client
	mu            sync.Mutex
}

type ReplicationMessage struct {
	SenderId            int                     `json:"senderid"`
	Dest                int                     `json:"receiverid"`
	IntendedRecipientId int                     `json:"intendedreceipientid"`
	DatabaseMessage     redisDB.DatabaseMessage `json:"databasemessage"`
	MessageCode         int                     `json:"messagecode"`
	// 0 - failed write
	// 1 - ACK write
	// 2 - replication data
	// 3 - hinted handoff
	// 4 -failed read
	// 5 - ACK read
	// 6 - key data
	// 7 - commit
	// 8 - ACK commit
	// 9 - ACK handoff response
	// 10 - handoff commit
}

func (r *Replicator) SendReplicationMessage(httpClient *http.Client, msg ReplicationMessage) (failedConn int, failedTarget int) {
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
func (r *Replicator) ReplicateWrites(data redisDB.DatabaseMessage) bool {
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
		msg := ReplicationMessage{SenderId: id, Dest: target, DatabaseMessage: data, MessageCode: 2}
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
	if failedConnections > 0 {
		for {
			if failedConnections == 0 || startingPt == id {
				break
			}
			newTargets := r.getNodes(failedConnections, startingPt)
			for _, target := range newTargets {
				// sending handoff mesages
				handoffMsg := ReplicationMessage{SenderId: id, Dest: target, IntendedRecipientId: failedTargets[count], DatabaseMessage: data, MessageCode: 3}
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
			fmt.Println("looping here number of failed connections" + strconv.Itoa(failedConnections))
		}
	}
	time.Sleep(time.Second * 1)
	fmt.Println("Results for write replication")
	quorumRes := r.checkWriteQuorum()
	if quorumRes {
		// write quorum passed, send values to nodes
		fmt.Println("sending values to nodes that were contactable")
		for targetId, res := range r.WriteCheck {
			if res {
				// sends values to these nodes
				msg := ReplicationMessage{SenderId: id, Dest: targetId, DatabaseMessage: data, MessageCode: 7}
				res, t := r.SendReplicationMessage(r.HttpClient, msg)
				if res == 1 {
					fmt.Println("Message to " + strconv.Itoa(t) + " failed to connect")
				}
			}
		}
		for targetId, res := range r.HandoffCheck {
			if res {
				msg := ReplicationMessage{SenderId: id, Dest: targetId, DatabaseMessage: data, MessageCode: 10}
				res, t := r.SendReplicationMessage(r.HttpClient, msg)
				if res == 1 {
					fmt.Println("Message to " + strconv.Itoa(t) + " failed to connect")
				}
			}
		}
	}
	fmt.Println("Write quorum results is :" + strconv.FormatBool(quorumRes))
	return quorumRes
}

// starts replication processs
// needs to be updated to use node structure from gossip
func (r *Replicator) ReplicateReads(key redisDB.DatabaseMessage) (bool, string) {
	fmt.Println("Node: " + (r.Id) + " begining replication")
	id, err := strconv.Atoi(r.Id)
	if err != nil {
		fmt.Println(err)
	}
	r.ReadCheck = make(map[int]string)
	targets := r.getNodes(r.N-1, id+1)
	// checks N-1 nodes in preference list
	for _, target := range targets {
		msg := ReplicationMessage{SenderId: id, Dest: target, DatabaseMessage: key, MessageCode: 6}
		res, t := r.SendReplicationMessage(r.HttpClient, msg)
		if res == 1 {
			fmt.Println("Message to " + strconv.Itoa(t) + " failed")
		}
	}
	// sents to nodes next in priority list
	time.Sleep(time.Second * 1)
	fmt.Println("Results for read replication")
	fmt.Println(r.ReadCheck)
	res, val := r.checkReadQuorum()
	fmt.Println(res)
	return res, val
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
	// handoff nodes considered when checking against W
	for _, v := range r.HandoffCheck {
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

func (r *Replicator) checkReadQuorum() (bool, string) {
	fmt.Println("in check write quorum")
	fmt.Println(r.ReadCheck)
	values := make(map[string]int)
	r.mu.Lock()
	for _, v := range r.ReadCheck {
		if values[v] == 0 {
			values[v] = 1
		} else {
			values[v] = values[v] + 1
		}
	}
	max := 0
	key := ""
	for k, count := range values {
		if max < count {
			max = count
			key = k
		}
	}
	r.mu.Unlock()
	fmt.Println(max)
	if max < r.R {
		fmt.Println("failed write quorum")
		return false, key
	} else {
		fmt.Println("passed write quorum")
		return true, key
	}
}

// sends response to handoff message
func (r *Replicator) HandleHandoff(msg ReplicationMessage) {

	replyMsg := ReplicationMessage{SenderId: getOwnId(), Dest: msg.SenderId, MessageCode: 9}
	// send message to indicate handoff message received
	go r.SendReplicationMessage(r.HttpClient, replyMsg)

}

func (r *Replicator) HandleHandoffCommit(msg ReplicationMessage) {
	data := msg.DatabaseMessage
	ctx := context.Background()
	rdb := r.Rdb
	err := rdb.Set(ctx, data.Key, data.Value, 0).Err()
	if err != nil {
		panic(err)
	}
	msgToIntended := ReplicationMessage{SenderId: getOwnId(), Dest: msg.IntendedRecipientId, DatabaseMessage: msg.DatabaseMessage, MessageCode: 2}
	for {
		res, _ := r.SendReplicationMessage(r.HttpClient, msgToIntended)
		if res == 0 {
			break
		}
		// tries to send again in 1 second
		time.Sleep(time.Second * 1)
	}
	// delete handoff data from node
	rdb.Del(ctx, data.Key)
	fmt.Println("successfully sent handoff data")
}

func (r *Replicator) HandleWriteResponse(receivedMsg ReplicationMessage) {
	msg := ReplicationMessage{SenderId: getOwnId(), Dest: receivedMsg.SenderId, MessageCode: 1}
	go r.SendReplicationMessage(r.HttpClient, msg)
}

func (r *Replicator) HandleReadResponse(receivedMsg ReplicationMessage) {
	// need add part for writes to redis
	// successful write to redis
	r.mu.Lock()
	data := receivedMsg.DatabaseMessage
	ctx := context.Background()
	rdb := r.Rdb
	val, err := rdb.Get(ctx, data.Key).Result()
	if err != nil {
		panic(err)
	}
	msg := ReplicationMessage{SenderId: getOwnId(), Dest: receivedMsg.SenderId, DatabaseMessage: redisDB.DatabaseMessage{Key: data.Key, Value: val}, MessageCode: 5}
	go r.SendReplicationMessage(r.HttpClient, msg)
	r.mu.Unlock()
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

func (r *Replicator) AddSuccessfulRead(msg ReplicationMessage) {
	r.mu.Lock()
	r.ReadCheck[msg.SenderId] = msg.DatabaseMessage.Value
	r.mu.Unlock()
	fmt.Println(r.ReadCheck)
}

func (r *Replicator) AddSuccessfulHandoff(id int) {
	r.mu.Lock()
	r.HandoffCheck[id] = true
	r.mu.Lock()
	fmt.Println(r.HandoffCheck)
}

func (r *Replicator) HandleCommit(msg ReplicationMessage) {
	r.mu.Lock()
	data := msg.DatabaseMessage
	ctx := context.Background()
	rdb := r.Rdb
	err := rdb.Set(ctx, data.Key, data.Value, 0).Err()
	if err != nil {
		panic(err)
	}
	// sends response to indicate successful commit
	replyMsg := ReplicationMessage{SenderId: getOwnId(), Dest: msg.SenderId, MessageCode: 8}
	go r.SendReplicationMessage(r.HttpClient, replyMsg)
	r.mu.Unlock()
}
