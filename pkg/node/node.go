package node

import (
	"ShoppiDB/pkg/byzantine"
	replication "ShoppiDB/pkg/data_replication"
	gossip "ShoppiDB/pkg/gossip"
	redisDB "ShoppiDB/pkg/redisDB"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/k0kubun/pp/v3"
)

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
			// failed write
			fmt.Println("Received from Node: " + strconv.Itoa(msg.SenderId) + " failed write")
		}
	case 1:
		{
			// success write
			fmt.Println("Received from Node: " + strconv.Itoa(msg.SenderId) + " successful write")
			n.Replicator.AddSuccessfulWrite(msg.SenderId)
		}
	case 2:
		{
			// response
			fmt.Println("Received from Node: " + strconv.Itoa(msg.SenderId) + " replication response")
			go n.Replicator.HandleWriteResponse(msg)
		}
	case 3:
		{
			// hinted handoff
			fmt.Println("Received from Node: " + strconv.Itoa(msg.SenderId) + " handoff data")
			go n.Replicator.HandleHandoff(msg)
		}
	case 4:
		{
			// failed read
			fmt.Println("Received from Node: " + strconv.Itoa(msg.SenderId) + " failed read")
		}
	case 5:
		{
			fmt.Println("Received from Node: " + strconv.Itoa(msg.SenderId) + " successful read")
			n.Replicator.AddSuccessfulRead(msg.SenderId)
		}
	case 6:
		{
			fmt.Println("Received from Node: " + strconv.Itoa(msg.SenderId) + " key data")
			go n.Replicator.HandleReadResponse(msg)
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

func getHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	fmt.Println("Request for GET function")
	w.Header().Set("Content-Type", "application/json")
	if r.Body == nil {
		http.Error(w, "Please send a request body", 400)
		return
	}
	var message redisDB.DatabaseMessage
	err := json.NewDecoder(r.Body).Decode(&message)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	ctx := context.Background()
	rdb := redisDB.GetDBClient()
	val, err := rdb.Get(ctx, message.Key).Result()
	if err != nil {
		panic(err)
	}
	fmt.Println("key: ", val)
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(val)
}

func putHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	fmt.Println("Request for PUT function")
	w.Header().Set("Content-Type", "application/json")
	if r.Body == nil {
		http.Error(w, "Please send a request body", 400)
		return
	}
	var message redisDB.DatabaseMessage
	err := json.NewDecoder(r.Body).Decode(&message)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	ctx := context.Background()
	rdb := redisDB.GetDBClient()
	err = rdb.Set(ctx, message.Key, message.Value, 0).Err()
	if err != nil {
		panic(err)
	}
	fmt.Println(message)
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(message)
}

func (n *Node) StartHTTPServer() {
	fmt.Println("Starting HTTP Server")
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", defaultHandler).Methods("GET")
	router.HandleFunc("/byzantine", byzantineHandler).Methods("POST")
	router.HandleFunc("/replication", n.replicationHandler).Methods("POST")
	router.HandleFunc("/gossip", n.gossipHandler).Methods("POST")
	router.HandleFunc("/get", getHandler).Methods("POST")
	router.HandleFunc("/put", putHandler).Methods("POST")
	log.Fatal(http.ListenAndServe(":8080", handlers.CORS()(router)))
}

/*
	Clients and Transports are safe for concurrent use by multiple goroutines and for efficiency should only be created once and re-used.
*/
func GetHTTPClient() *http.Client {
	tr := &http.Transport{
		MaxIdleConns:       100,
		IdleConnTimeout:    30 * time.Second,
		MaxConnsPerHost:    100,
		DisableCompression: true,
	}
	client := &http.Client{
		Timeout:   300 * time.Millisecond,
		Transport: tr,
	}
	return client
}

func (n *Node) BasicHTTPGET(nodeId string, httpClient *http.Client) {
	pp.Println("To be sending message to " + nodeId)
	req, err := http.NewRequest("GET", "http://"+nodeId+":8080/", nil)
	checkErr(err)
	resp, err := httpClient.Do(req)
	checkErr(err)
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	checkErr(err)
	pp.Println(string(b))
	time.Sleep(time.Second * 5)
}

func (n *Node) DbPUT(nodeId string, httpClient *http.Client, value string) {
	pp.Println("To be sending message to " + nodeId)
	msg := redisDB.DatabaseMessage{Key: "key", Value: value}
	msgJson, err := json.Marshal(msg)
	checkErr(err)
	req, err := http.NewRequest(http.MethodPost, "http://"+nodeId+":8080/put", bytes.NewBuffer(msgJson))
	checkErr(err)
	resp, err := httpClient.Do(req)
	checkErr(err)
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	checkErr(err)
	pp.Println(string(b))
	time.Sleep(time.Second * 5)
}

func (n *Node) DbGET(nodeId string, httpClient *http.Client) {
	pp.Println("To be sending message to " + nodeId)
	msg := redisDB.DatabaseMessage{Key: "key"}
	msgJson, err := json.Marshal(msg)
	checkErr(err)
	req, err := http.NewRequest(http.MethodPost, "http://"+nodeId+":8080/get", bytes.NewBuffer(msgJson))
	checkErr(err)
	resp, err := httpClient.Do(req)
	checkErr(err)
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	checkErr(err)
	pp.Println(string(b))
	time.Sleep(time.Second * 5)
}

func checkErr(err error) {
	if err != nil {
		fmt.Println("ERROR")
		fmt.Println(err)
	}
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
* @return a nested array consisting of the assigned token set
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
