package http_api

import (
	"ShoppiDB/pkg/byzantine"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
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

func StartHTTPServer() {
	fmt.Println("Starting HTTP Server")
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/byzantine", byzantineHandler).Methods("POST")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func GetHTTPClient() *http.Client {
	client := &http.Client{Timeout: 10 * time.Second}
	return client
}
