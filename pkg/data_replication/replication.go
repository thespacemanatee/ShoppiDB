package replication

import (
	"ShoppiDB/pkg/data_versioning"
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
)

type Replicator struct {
	Id           string
	N            int
	R            int
	W            int
	Queue        *PriorityQueue
	WriteCheck   map[int]bool
	HandoffCheck map[int]bool
	ReadCheck    map[int]string
	mu           sync.Mutex
	queueLock    sync.Mutex
}

type ReplicationMessage struct {
	SenderId            int                        `json:"senderid"`
	Dest                int                        `json:"receiverid"`
	IntendedRecipientId int                        `json:"intendedreceipientid"`
	DataObject          data_versioning.DataObject `json:"dataobject"`
	MessageCode         int                        `json:"messagecode"`
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

type ClientReq struct {
	Timestamp  time.Time
	IsWriteReq bool
	DataObject data_versioning.DataObject

	NodeStructure map[int]int
	resChannel    *chan Result
}

type Result struct {
	DataObject data_versioning.DataObject
	success    bool
}

func (r *Replicator) AddRequest(nodeStructure map[int]int, dataObject data_versioning.DataObject, isWrite bool) Result {
	r.queueLock.Lock()
	c := make(chan Result, 10)
	req := ClientReq{Timestamp: time.Now(), IsWriteReq: isWrite, DataObject: dataObject, resChannel: &c, NodeStructure: nodeStructure}
	head := r.Queue.Front()
	r.Queue.Push(&Item{Req: req})
	r.queueLock.Unlock()
	if head.Timestamp.Equal(time.Time{}) {
		//queue is empty
		if req.IsWriteReq {
			go r.ReplicateWrites(req)
		} else {
			go r.ReplicateReads(req)
		}
	}
	res := <-c
	fmt.Println("RECEIVED RESPONSE FROM REPLICATION")
	fmt.Println(res)
	return res
}

func (r *Replicator) HandleNextInQueue() {
	r.queueLock.Lock()
	r.Queue.Pop()
	head := r.Queue.Front()
	r.queueLock.Unlock()
	if !head.Timestamp.Equal(time.Time{}) {
		//queue is not empty
		if head.IsWriteReq {
			go r.ReplicateWrites(head)
		} else {
			go r.ReplicateReads(head)
		}
	} else {
		fmt.Println("queue is empty")
	}
}

// starts write replication processs
func (r *Replicator) ReplicateWrites(req ClientReq) {
	fmt.Println("Node: " + (r.Id) + " beginning write replication")
	id, err := strconv.Atoi(r.Id)
	if err != nil {
		fmt.Println(err)
	}
	r.WriteCheck = make(map[int]bool)
	r.HandoffCheck = make(map[int]bool)
	failedConnections := 0
	failedTargets := []int{}
	targets := r.getNodes(r.N-1, 1, req.NodeStructure)
	// contact N-1 nodes in preferenceList
	for _, target := range targets {
		msg := ReplicationMessage{SenderId: id, Dest: target, DataObject: req.DataObject, MessageCode: 0}
		httpClient := GetHTTPClient(500 * time.Millisecond)
		res, t := r.SendReplicationMessage(httpClient, msg)
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
		//for {
		// if failedConnections == 0 || startingPt == id {
		// 	break
		// }
		newTargets := r.getNodes(failedConnections, startingPt, req.NodeStructure)
		for _, target := range newTargets {
			// sending handoff mesages
			handoffMsg := ReplicationMessage{SenderId: id, Dest: target, IntendedRecipientId: failedTargets[count], DataObject: req.DataObject, MessageCode: 5}
			httpClient := GetHTTPClient(500 * time.Millisecond)
			res, _ := r.SendReplicationMessage(httpClient, handoffMsg)
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
		//}
	}
	time.Sleep(time.Second * 1)
	fmt.Println("Results for write replication")
	quorumRes := r.checkWriteQuorum()
	// sending of results back to client
	*req.resChannel <- Result{req.DataObject, quorumRes}
	if quorumRes {
		// write quorum passed, send values to nodes
		fmt.Println("sending values to nodes that were contactable")
		for targetId, res := range r.WriteCheck {
			if res {
				// sends values to these nodes
				msg := ReplicationMessage{SenderId: id, Dest: targetId, DataObject: req.DataObject, MessageCode: 2}
				httpClient := GetHTTPClient(1 * time.Second)
				res, t := r.SendReplicationMessage(httpClient, msg)
				// failed to send over data
				if res == 1 {
					fmt.Println("Message to " + strconv.Itoa(t) + " failed to connect")
					httpClient := GetHTTPClient(1 * time.Second)

					go r.HandleFailedSend(httpClient, msg)
				}
			}
		}
		// send data to handoff node
		count := 0
		for targetId, toSend := range r.HandoffCheck {
			if toSend {
				msg := ReplicationMessage{SenderId: id, Dest: targetId, IntendedRecipientId: failedTargets[count], DataObject: req.DataObject, MessageCode: 5}
				httpClient := GetHTTPClient(1 * time.Second)
				res, t := r.SendReplicationMessage(httpClient, msg)
				if res == 1 {
					fmt.Println("Message to " + strconv.Itoa(t) + " failed to connect")
					go r.HandleFailedSend(httpClient, msg)
				}
				count++
			}
		}
	}
	fmt.Println("Write quorum results is :" + strconv.FormatBool(quorumRes))
	go r.HandleNextInQueue()
}

// starts read replication processs
func (r *Replicator) ReplicateReads(req ClientReq) (bool, string) {
	fmt.Println("Node: " + (r.Id) + " beginning read replication")
	id, err := strconv.Atoi(r.Id)
	if err != nil {
		fmt.Println(err)
	}
	r.ReadCheck = make(map[int]string)
	targets := r.getNodes(r.N-1, 1, req.NodeStructure)
	// checks N-1 nodes in preference list
	for _, target := range targets {
		msg := ReplicationMessage{SenderId: id, Dest: target, DataObject: req.DataObject, MessageCode: 7}
		httpClient := GetHTTPClient(500 * time.Millisecond)
		res, t := r.SendReplicationMessage(httpClient, msg)
		if res == 1 {
			fmt.Println("Message to " + strconv.Itoa(t) + " failed")
		}
	}
	// sents to nodes next in priority list
	time.Sleep(time.Second * 1)
	fmt.Println("Results for read replication")
	res, val := r.checkReadQuorum()
	// sending read result back to client
	*req.resChannel <- Result{DataObject: data_versioning.DataObject{Key: req.DataObject.Key, Value: val, Context: req.DataObject.Context}, success: res}
	go r.HandleNextInQueue()
	return res, val
}
func (r *Replicator) SendReplicationMessage(httpClient *http.Client, msg ReplicationMessage) (failedConn int, failedTarget int) {
	// used to indicate whether connection failed
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("panic occured fail to connect")
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
	resp, err := httpClient.Do(req)
	checkErr(err)
	defer resp.Body.Close()
	var receivedMessage string
	json.NewDecoder(resp.Body).Decode(&receivedMessage)
	return 0, 0
}

func (r *Replicator) checkWriteQuorum() bool {
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
	if len(values) < r.W {
		fmt.Println("failed write quorum")
		return false
	} else {
		fmt.Println("passed write quorum")
		return true
	}
}

func (r *Replicator) checkReadQuorum() (bool, string) {
	values := make(map[string]int)
	r.mu.Lock()
	fmt.Println(r.ReadCheck)
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
	if max < r.R {
		fmt.Println("failed read quorum")
		return false, key
	} else {
		fmt.Println("passed read quorum")
		return true, key
	}
}

// reply message to indicate node alive
func (r *Replicator) HandleWriteResponse(receivedMsg ReplicationMessage) {
	msg := ReplicationMessage{SenderId: getOwnId(), Dest: receivedMsg.SenderId, MessageCode: 1}
	httpClient := GetHTTPClient(500 * time.Millisecond)
	go r.SendReplicationMessage(httpClient, msg)
}

// reply message to indicate node alive
func (r *Replicator) HandleReadResponse(receivedMsg ReplicationMessage) {
	// need add part for writes to redis
	// successful write to redis
	r.mu.Lock()
	rdb := redisDB.GetDBClient()
	data := receivedMsg.DataObject
	ctx := context.Background()
	valJson, err := rdb.Get(ctx, data.Key).Result()
	r.mu.Unlock()
	if err != nil {
		panic(err)
	}
	var val data_versioning.DataObject
	json.Unmarshal([]byte(valJson), &val)
	msg := ReplicationMessage{SenderId: getOwnId(), Dest: receivedMsg.SenderId, DataObject: val, MessageCode: 8}
	httpClient := GetHTTPClient(500 * time.Millisecond)
	go r.SendReplicationMessage(httpClient, msg)
}

// sends response to handoff message
func (r *Replicator) HandleHandoffResponse(msg ReplicationMessage) {
	replyMsg := ReplicationMessage{SenderId: getOwnId(), Dest: msg.SenderId, MessageCode: 4}
	// send message to indicate handoff node alive
	httpClient := GetHTTPClient(500 * time.Millisecond)
	go r.SendReplicationMessage(httpClient, replyMsg)
}

// commits data to database
func (r *Replicator) HandleCommit(msg ReplicationMessage) {
	data := msg.DataObject
	dataJson, err := json.Marshal(data)
	checkErr(err)

	ctx := context.Background()
	rdb := redisDB.GetDBClient()
	err = rdb.Set(ctx, data.Key, dataJson, 0).Err()
	fmt.Println(data)
	if err != nil {
		panic(err)
	}
}

// commits data to handoff database
// repeatedly trys to send to intended node
// deletes handoff data upon successful sending to intended
func (r *Replicator) HandleHandoffCommit(msg ReplicationMessage) {
	data := msg.DataObject
	dataJson, err := json.Marshal(data)
	checkErr(err)
	ctx := context.Background()
	rdb := redisDB.GetDBClient()
	err = rdb.Set(ctx, data.Key, dataJson, 0).Err()
	if err != nil {
		panic(err)
	}
	msgToIntended := ReplicationMessage{SenderId: getOwnId(), Dest: msg.IntendedRecipientId, DataObject: msg.DataObject, MessageCode: 6}
	for {
		httpClient := GetHTTPClient(500 * time.Millisecond)
		res, _ := r.SendReplicationMessage(httpClient, msgToIntended)
		if res == 0 {
			break
		}
		// tries to send again in 2 second
		time.Sleep(time.Second * 2)
	}
	// delete handoff data from node
	rdb.Del(ctx, data.Key)
	fmt.Println("successfully sent handoff data")
}

// repeatedly sends msg every 1s
func (r *Replicator) HandleFailedSend(httpClient *http.Client, msg ReplicationMessage) {
	for {
		res, t := r.SendReplicationMessage(httpClient, msg)
		if res == 0 {
			fmt.Println("Message to " + strconv.Itoa(t) + " send over")
			return
		}
		time.Sleep(time.Second * 1)
	}
}

// commits data from handoff node
func (r *Replicator) HandleHandoffToIntended(msg ReplicationMessage) {
	data := msg.DataObject
	dataJson, err := json.Marshal(data)
	checkErr(err)
	ctx := context.Background()
	rdb := redisDB.GetDBClient()
	err = rdb.Set(ctx, data.Key, dataJson, 0).Err()
	if err != nil {
		panic(err)
	}
}

// stores response from node
func (r *Replicator) AddSuccessfulWrite(id int) {
	r.mu.Lock()
	r.WriteCheck[id] = true
	r.mu.Unlock()
}

// stores response from node
func (r *Replicator) AddSuccessfulRead(msg ReplicationMessage) {
	r.mu.Lock()
	r.ReadCheck[msg.SenderId] = msg.DataObject.Value
	r.mu.Unlock()
}

// stores response from handoff node
func (r *Replicator) AddSuccessfulHandoff(id int) {
	r.mu.Lock()
	r.HandoffCheck[id] = true
	r.mu.Unlock()
}

func getOwnId() int {
	id := os.Getenv("NODE_ID")
	idNum, err := strconv.Atoi(id)
	checkErr(err)
	return idNum
}

func checkErr(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

//function returns the node ids to send for replication
func (r *Replicator) getNodes(numNodes int, startingPt int, nodeStructure map[int]int) []int {
	nodesTosend := make([]int, numNodes)
	count := 0
	for i := startingPt; i < startingPt+numNodes; i++ {
		nodesTosend[count] = nodeStructure[i%len(nodeStructure)]
		count++
	}
	return nodesTosend
}

func GetHTTPClient(timeout time.Duration) *http.Client {
	tr := &http.Transport{
		MaxIdleConns:       100,
		IdleConnTimeout:    30 * time.Second,
		MaxConnsPerHost:    100,
		DisableCompression: true,
	}
	client := &http.Client{
		Timeout:   timeout,
		Transport: tr,
	}
	return client
}
