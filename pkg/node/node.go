package node

import (
	"ShoppiDB/pkg/byzantine"
	replication "ShoppiDB/pkg/data_replication"
	"ShoppiDB/pkg/data_versioning"
	"ShoppiDB/pkg/gossip"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"math"
	"math/big"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type GetRequest struct {
	Key *string `json:"key"`
}

type PutRequest struct {
	Key     *string                      `json:"key"`
	Value   *string                      `json:"value"`
	Context *data_versioning.VectorClock `json:"context"`
}

type Node struct {
	nonce         []string
	ContainerName string
	TokenSet      [][2]int
	Membership    bool
	Replicator    *replication.Replicator
	Gossiper      gossip.Gossip

	// IsSeed            bool
	// NodeRingPositions []int
	// Ring              *conHashing.Ring
}

func (n *Node) updateNonce(nonce string) {
	n.nonce = append(n.nonce, nonce)
	fmt.Println("Appended Nonce")
	fmt.Println(n.nonce)
}

func (n *Node) gossipHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Body == nil {
		http.Error(w, "Please send a request body", 400)
		return
	}
	var msg gossip.GossipMessage
	err := json.NewDecoder(r.Body).Decode(&msg)
	httpCheckErr(w, err)
	response := n.Gossiper.CompareAndUpdate(msg)

	json.NewEncoder(w).Encode(response) //Writing the message back
}

func (n *Node) replicationHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Body == nil {
		http.Error(w, "Please send a request body", 400)
		return
	}
	var msg replication.ReplicationMessage
	err := json.NewDecoder(r.Body).Decode(&msg)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	fmt.Println(msg)
	w.WriteHeader(http.StatusAccepted)
	switch msg.MessageCode {
	case 0:
		{
			// coordinator node want to write on current node
			fmt.Println("Received from Node: " + strconv.Itoa(msg.SenderId) + " prep recv")
			go n.Replicator.HandleWriteResponse(msg)
		}
	case 1:
		{
			// successful response from node write
			fmt.Println("Received from Node: " + strconv.Itoa(msg.SenderId) + " ack prep recv")
			n.Replicator.AddSuccessfulWrite(msg.SenderId)
		}
	case 2:
		{
			// committing data
			fmt.Println("Received from Node: " + strconv.Itoa(msg.SenderId) + " commit write data")
			go n.Replicator.HandleCommit(msg)
		}
	case 3:
		{
			// prep recv hinted handoff
			fmt.Println("Received from Node: " + strconv.Itoa(msg.SenderId) + " prep recv handoff data")
			go n.Replicator.HandleHandoffResponse(msg)
		}
	case 4:
		{
			// ack for recv hinted handoff
			fmt.Println("Received from Node: " + strconv.Itoa(msg.SenderId) + " ack prep recv handoff data")
			n.Replicator.AddSuccessfulHandoff(msg.SenderId)
		}
	case 5:
		{
			// committing to handoff node
			fmt.Println("Received from Node: " + strconv.Itoa(msg.SenderId) + " commit write data to handoff")
			go n.Replicator.HandleHandoffCommit(msg)
		}
	case 6:
		{
			// handoff node to intended node
			fmt.Println("Received from Node: " + strconv.Itoa(msg.SenderId) + " data from handoff node to intended")
			go n.Replicator.HandleHandoffToIntended(msg)
		}
	case 7:
		{
			// coordinator node want to read on current node
			fmt.Println("Received from Node: " + strconv.Itoa(msg.SenderId) + " read key data")
			go n.Replicator.HandleReadResponse(msg)

		}
	case 8:
		{
			fmt.Println("Received from Node: " + strconv.Itoa(msg.SenderId) + " successful read data")
			go n.Replicator.AddSuccessfulRead(msg)
		}

	default:
		{
			fmt.Println("Wrong message code used")
		}
	}
}

func byzantineHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Body == nil {
		http.Error(w, "Please send a request body", 400)
		return
	}
	var byzantineMessage byzantine.ByzantineMessage
	err := json.NewDecoder(r.Body).Decode(&byzantineMessage)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	// Business logic
	fmt.Println(byzantineMessage)
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(byzantineMessage) //Writing the message back
}

func defaultHandler(w http.ResponseWriter, r *http.Request) {
	id := os.Getenv("NODE_ID")
	fmt.Fprintln(w, "U have called node "+id+", The path is:", html.EscapeString(r.URL.Path))
}

func checkHeartbeat(w http.ResponseWriter, r *http.Request) {
	id := os.Getenv("NODE_ID")
	fmt.Fprintln(w, "U have called node "+id+", The path is:", html.EscapeString(r.URL.Path))
}

func (n *Node) getHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	fmt.Println("Request for GET function")
	w.Header().Set("Content-Type", "application/json")
	if r.Body == nil {
		http.Error(w, "Please send a request body", 400)
		return
	}
	var message GetRequest
	err := json.NewDecoder(r.Body).Decode(&message)
	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), 400)
		return
	}
	//hashKey := consistent_hashing.GetMD5Hash(*message.Key)
	//nodeStructure := n.GetPreferenceList(*hashKey)
	nodeStructure := map[int]int{1: 2, 2: 3, 3: 4, 4: 5, 5: 6, 6: 7, 7: 8, 8: 9, 9: 0}
	// not sure where to get context for DataObject
	res := n.Replicator.AddRequest(nodeStructure, data_versioning.DataObject{Key: *message.Key} ,false)
	if res.Success {
		w.WriteHeader(http.StatusAccepted)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
	json.NewEncoder(w).Encode(res.ConflictingObjects)
}

func (n *Node) putHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	fmt.Println("Request for PUT function")
	w.Header().Set("Content-Type", "application/json")
	if r.Body == nil {
		http.Error(w, "Please send a request body", 400)
		return
	}

	var message PutRequest
	err := json.NewDecoder(r.Body).Decode(&message)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	newObject := &data_versioning.DataObject{
		Key:   *message.Key,
		Value: *message.Value,
	}
	if message.Context == nil {
		fmt.Println("MESSAGE NO CLOCK!")
		clock := data_versioning.NewVectorClock(os.Getenv("NODE_ID"))
		newObject.Context = clock
	} else {
		fmt.Println("MESSAGE GOT CLOCK!")
		newObject.Context = *message.Context
		data_versioning.UpdateVectorClock(os.Getenv("NODE_ID"), &newObject.Context)
	}

	// hashKey := consistent_hashing.GetMD5Hash(*message.Key)
	// nodeStructure := n.GetPreferenceList(*hashKey)
	nodeStructure := map[int]int{1: 2, 2: 3, 3: 4, 4: 5, 5: 6, 6: 7, 7: 8, 8: 9, 9: 0}
	// not sure where to get context for DataObject
	res := n.Replicator.AddRequest(nodeStructure, data_versioning.DataObject{Key: *message.Key, Value: *message.Value, Context: newObject.Context}, true)
	if res.Success {
		w.WriteHeader(http.StatusAccepted)
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
	json.NewEncoder(w).Encode(res.DataObject)
}

func (n *Node) StartHTTPServer() {
	fmt.Println("Starting HTTP Server")
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", defaultHandler).Methods("GET")
	router.HandleFunc("/checkHeartbeat", checkHeartbeat).Methods("GET")
	router.HandleFunc("/byzantine", byzantineHandler).Methods("POST")
	router.HandleFunc("/replication", n.replicationHandler).Methods("POST")
	router.HandleFunc("/gossip", n.gossipHandler).Methods("POST")
	router.HandleFunc("/get", n.getHandler).Methods("POST")
	router.HandleFunc("/put", n.putHandler).Methods("POST")
	log.Fatal(http.ListenAndServe(":8080", handlers.CORS()(router)))
}

/*
	Clients and Transports are safe for concurrent use by multiple goroutines and for efficiency should only be created once and re-used.
*/
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

func httpCheckErr(w http.ResponseWriter, err error) {
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}
func getNodeTotal() int {
	i, err := strconv.Atoi(os.Getenv("NODE_TOTAL"))
	checkErr(err)
	return i
}

/**
* Returns a nested array consisting of the assigned token set
* Eg. [[1,2], [4,5],[24,25]]
*
*
* @return a nested a	rray consisting of the assigned token set
 */

func GenTokenSet() [][2]int {
	var tokenSet [][2]int
	numNodes := getNodeTotal()
	numTokens := math.Floor(64 / float64(numNodes))
	for i := 0; i < int(numTokens); i++ {
		rand.Seed(time.Now().UnixNano())
		min := 0
		max := 63
		randInt := rand.Intn(max-min) + min
		// check if the token is alr assigned
		dup := false
		for {
			for _, t := range tokenSet {
				if randInt == t[0] {
					dup = true
					randInt = rand.Intn(max-min) + min
				}
			}
			if dup != true {
				break
			}
		}

		if randInt < 63 {
			tokenSet = append(tokenSet, [2]int{randInt, randInt + 1})
		} else {
			tokenSet = append(tokenSet, [2]int{randInt, 0})
		}
	}
	return tokenSet
}

// returns preference list
// map of all nodes where key is order of preference and value is id of node
func (n *Node) GetPreferenceList(hashKey big.Float) map[int]int {
	hashKeyInt64, _ := hashKey.Int64()
	preferenceList := make(map[int]int)

	nodeMap := n.Gossiper.CommNodeMap // to check if it is a phy node
	startingHashRange := [2]int{int(hashKeyInt64), int(hashKeyInt64) + 1}
	vnMap := n.Gossiper.VirtualNodeMap // get node based on hash val
	nodeMapIds := make(map[int]bool)
	for _, n := range nodeMap {
		if !n.StatusDown {
			nodeMapIds[n.Id] = true
		}
	}
	primaryNode := vnMap[startingHashRange]
	count := 0
	// iterate thru nodemapids starting from primarynode id +1 to 9 which is max node id
	for start := primaryNode.Id + 1; start < 10; start++ {
		// if node id exists in node map add to preference list
		if nodeMapIds[start] {
			preferenceList[count] = start
			count++
		}
	}
	//iterate thru nodemapids starting from node 0 to primarynode id
	for start := 0; start < primaryNode.Id; start++ {
		if nodeMapIds[start] {
			preferenceList[count] = start
			count++
		}
	}
	return preferenceList
}

func getHashRange(hashVal int64, totalNumNodes int) [2]int {
	fmt.Print("this is the hashKey")
	fmt.Println(hashVal)
	firstVal := int(hashVal * int64(totalNumNodes))
	secondVal := firstVal + 1
	return [2]int{firstVal, secondVal}
}

func hashValueContains(s []int, e int64) bool {
	for _, a := range s {
		if a == int(e) {
			return true
		}
	}
	return false
}

func checkErr(err error) {
	if err != nil {
		fmt.Println("ERROR")
		fmt.Println(err)
	}
}
