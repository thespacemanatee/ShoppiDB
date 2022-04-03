package http_api

import (
	"ShoppiDB/pkg/byzantine"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/k0kubun/pp/v3"
)

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

func StartHTTPServer() {
	fmt.Println("Starting HTTP Server")
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", defaultHandler).Methods("GET")
	router.HandleFunc("/byzantine", byzantineHandler).Methods("POST")
	log.Fatal(http.ListenAndServe(":8080", router))
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
		Timeout:   10 * time.Second,
		Transport: tr,
	}
	return client
}

func BasicHTTPGET(nodeId string, httpClient *http.Client) {
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

func checkErr(err error) {
	if err != nil {
		fmt.Println("ERROR")
		fmt.Println(err)
	}
}
