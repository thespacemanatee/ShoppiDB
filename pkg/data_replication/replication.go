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
	Id             string
	N              int
	R              int
	W              int
	NodeStructure  map[int]int
	WriteCheck     map[int]bool
	HandoffCheck   map[int]bool
	ReadCheck      map[int]string
	Rdb            redis.Client
	HttpClient     *http.Client
	LongerTOClient *http.Client
	mu             sync.Mutex
}

type ReplicationMessage struct {
	SenderId            int                     `json:"senderid"`
	Dest                int                     `json:"receiverid"`
	IntendedRecipientId int                     `json:"intendedreceipientid"`
	DatabaseMessage     redisDB.DatabaseMessage `json:"databasemessage"`
	MessageCode         int                     `json:"messagecode"`
	// 0 - prep recv
	// 1 - ack prep recv
	// 2 - commit write data
	// 3 - prep recv to handoff
	// 4 - ack prep recv to handoff
	// 5 - commit write data to handoff
	// 6 - commit write to intended from handoff
	// 7 - read
	// 8 - read response
}

// starts write replication processs
func (r *Replicator) ReplicateWrites(nodeStructure map[int]int, data redisDB.DatabaseMessage) bool {
	r.NodeStructure = nodeStructure
	fmt.Println("Node: " + (r.Id) + " begining replication")
	id, err := strconv.Atoi(r.Id)
	if err != nil {
		fmt.Println(err)
	}
	r.WriteCheck = make(map[int]bool)
	r.HandoffCheck = make(map[int]bool)
	failedConnections := 0
	failedTargets := []int{}
	targets := r.getNodes(r.N-1, 1)
	// contact N-1 nodes in preferenceList
	for _, target := range targets {
		msg := ReplicationMessage{SenderId: id, Dest: target, DatabaseMessage: data, MessageCode: 0}
		res, t := r.SendReplicationMessage(r.HttpClient, msg)
		if res == 1 {
			fmt.Println("Message to " + strconv.Itoa(t) + " failed")
			failedTargets = append(failedTargets, t)
		}
		failedConnections += res
	}
	startingPt := r.N
	count := 0
	// sents to nodes next in priority list based on number of failed conns
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
				msg := ReplicationMessage{SenderId: id, Dest: targetId, DatabaseMessage: data, MessageCode: 2}
				res, t := r.SendReplicationMessage(r.LongerTOClient, msg)
				// failed to send over data
				if res == 1 {
					fmt.Println("Message to " + strconv.Itoa(t) + " failed to connect")
					go r.HandleFailedSend(r.LongerTOClient, msg)
				}
			}
		}
		// send data to handoff node
		count := 0
		for targetId, toSend := range r.HandoffCheck {
			if toSend {
				msg := ReplicationMessage{SenderId: id, Dest: targetId, IntendedRecipientId: failedTargets[count], DatabaseMessage: data, MessageCode: 5}
				res, t := r.SendReplicationMessage(r.LongerTOClient, msg)
				if res == 1 {
					fmt.Println("Message to " + strconv.Itoa(t) + " failed to connect")
					go r.HandleFailedSend(r.LongerTOClient, msg)
				}
				count++
			}
		}
	}
	fmt.Println("Write quorum results is :" + strconv.FormatBool(quorumRes))
	return quorumRes
}

// starts read replication processs
func (r *Replicator) ReplicateReads(key redisDB.DatabaseMessage) (bool, string) {
	fmt.Println("Node: " + (r.Id) + " begining replication")
	id, err := strconv.Atoi(r.Id)
	if err != nil {
		fmt.Println(err)
	}
	r.ReadCheck = make(map[int]string)
	targets := r.getNodes(r.N-1, 1)
	// checks N-1 nodes in preference list
	for _, target := range targets {
		msg := ReplicationMessage{SenderId: id, Dest: target, DatabaseMessage: key, MessageCode: 7}
		res, t := r.SendReplicationMessage(r.HttpClient, msg)
		if res == 1 {
			fmt.Println("Message to " + strconv.Itoa(t) + " failed")
		}
	}
	// sents to nodes next in priority list
	time.Sleep(time.Second * 1)
	fmt.Println("Results for read replication")
	res, val := r.checkReadQuorum()
	fmt.Println(res)
	return res, val
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

func (r *Replicator) checkWriteQuorum() bool {
	fmt.Println("in check write quorum")
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

// reply message to indicate node alive
func (r *Replicator) HandleWriteResponse(receivedMsg ReplicationMessage) {
	msg := ReplicationMessage{SenderId: getOwnId(), Dest: receivedMsg.SenderId, MessageCode: 1}
	go r.SendReplicationMessage(r.HttpClient, msg)
}

// reply message to indicate node alive
func (r *Replicator) HandleReadResponse(receivedMsg ReplicationMessage) {
	// need add part for writes to redis
	// successful write to redis
	r.mu.Lock()
	data := receivedMsg.DatabaseMessage
	ctx := context.Background()
	rdb := r.Rdb
	val, err := rdb.Get(ctx, data.Key).Result()
	r.mu.Unlock()
	if err != nil {
		panic(err)
	}
	msg := ReplicationMessage{SenderId: getOwnId(), Dest: receivedMsg.SenderId, DatabaseMessage: redisDB.DatabaseMessage{Key: data.Key, Value: val}, MessageCode: 8}
	go r.SendReplicationMessage(r.HttpClient, msg)
}

// sends response to handoff message
func (r *Replicator) HandleHandoffResponse(msg ReplicationMessage) {
	replyMsg := ReplicationMessage{SenderId: getOwnId(), Dest: msg.SenderId, MessageCode: 4}
	// send message to indicate handoff node alive
	go r.SendReplicationMessage(r.HttpClient, replyMsg)
}

// commits data to database
func (r *Replicator) HandleCommit(msg ReplicationMessage) {
	r.mu.Lock()
	data := msg.DatabaseMessage
	ctx := context.Background()
	rdb := r.Rdb
	err := rdb.Set(ctx, data.Key, data.Value, 0).Err()
	r.mu.Unlock()
	if err != nil {
		panic(err)
	}
}

// commits data to handoff database
// repeatedly trys to send to intended node
// deletes handoff data upon successful sending to intended
func (r *Replicator) HandleHandoffCommit(msg ReplicationMessage) {
	data := msg.DatabaseMessage
	ctx := context.Background()
	rdb := r.Rdb
	err := rdb.Set(ctx, data.Key, data.Value, 0).Err()
	if err != nil {
		panic(err)
	}
	msgToIntended := ReplicationMessage{SenderId: getOwnId(), Dest: msg.IntendedRecipientId, DatabaseMessage: msg.DatabaseMessage, MessageCode: 6}
	for {
		res, _ := r.SendReplicationMessage(r.HttpClient, msgToIntended)
		if res == 0 {
			fmt.Print("why did res become ZERO")
			break
		}
		// tries to send again in 2 second
		time.Sleep(time.Second * 2)
	}
	// delete handoff data from node
	rdb.Del(ctx, data.Key)
	fmt.Println("successfully sent handoff data")
}

// repeatedly sends msg every 100 ms
func (r *Replicator) HandleFailedSend(httpClient *http.Client, msg ReplicationMessage) {
	for {
		res, t := r.SendReplicationMessage(httpClient, msg)
		if res == 0 {
			fmt.Println("Message to " + strconv.Itoa(t) + " send over")
			return
		}
		time.Sleep(time.Millisecond * 100)
	}
}

// commits data from handoff node
func (r *Replicator) HandleHandoffToIntended(msg ReplicationMessage) {
	data := msg.DatabaseMessage
	ctx := context.Background()
	rdb := r.Rdb
	err := rdb.Set(ctx, data.Key, data.Value, 0).Err()
	if err != nil {
		panic(err)
	}
}

// stores response from node
func (r *Replicator) AddSuccessfulWrite(id int) {
	r.mu.Lock()
	r.WriteCheck[id] = true
	fmt.Println(r.WriteCheck)
	r.mu.Unlock()
}

// stores response from node
func (r *Replicator) AddSuccessfulRead(msg ReplicationMessage) {
	r.mu.Lock()
	r.ReadCheck[msg.SenderId] = msg.DatabaseMessage.Value
	r.mu.Unlock()
	fmt.Println(r.ReadCheck)
}

// stores response from handoff node
func (r *Replicator) AddSuccessfulHandoff(id int) {
	r.mu.Lock()
	r.HandoffCheck[id] = true
	r.mu.Unlock()
	fmt.Println(r.HandoffCheck)
}

func getOwnId() int {
	id := os.Getenv("NODE_ID")
	idNum, err := strconv.Atoi(id)
	checkErr(err)
	return idNum
}

func checkErr(err error) {
	if err != nil {
		fmt.Println("CONNECTION ERROR")
		fmt.Println(err)
	}
}

//function returns the node ids to send for replication
func (r *Replicator) getNodes(numNodes int, startingPt int) []int {
	nodesTosend := make([]int, numNodes)
	count := 0
	for i := startingPt; i < startingPt+numNodes; i++ {
		nodesTosend[count] = r.NodeStructure[i%len(r.NodeStructure)]
		count++
	}
	return nodesTosend
}
