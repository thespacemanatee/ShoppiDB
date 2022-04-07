package node

import (
	"ShoppiDB/pkg/api"
	"ShoppiDB/pkg/byzantine"
	replication "ShoppiDB/pkg/data_replication"
	"ShoppiDB/pkg/gossip"
	"encoding/json"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"html"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Node struct {
	nonce         []string
	ContainerName string
	TokenSet      [][]int
	Membership    bool
	Replicator    *replication.Replicator
	Gossiper      gossip.Gossip
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

func (n *Node) StartHTTPServer() {
	fmt.Println("Starting HTTP Server")
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", defaultHandler).Methods("GET")
	router.HandleFunc("/byzantine", byzantineHandler).Methods("POST")
	router.HandleFunc("/replication", n.replicationHandler).Methods("POST")
	router.HandleFunc("/gossip", n.gossipHandler).Methods("POST")
	router.HandleFunc("/get", api.GetHandler).Methods("POST")
	router.HandleFunc("/put", api.PutHandler).Methods("POST")
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

func httpCheckErr(w http.ResponseWriter, err error) {
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
}
