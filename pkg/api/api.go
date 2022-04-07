package api

import (
	versioning "ShoppiDB/pkg/data_versioning"
	"ShoppiDB/pkg/redisDB"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type GetRequest struct {
	Key *string `json:"key"`
}

type PutRequest struct {
	Key     *string                 `json:"key"`
	Value   *string                 `json:"value"`
	Context *versioning.VectorClock `json:"context"`
}

func GetHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
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
	ctx := context.Background()
	rdb := redisDB.GetDBClient()
	fmt.Printf("Fetching key: %s from database\n", *message.Key)
	val, err := rdb.Get(ctx, *message.Key).Result()
	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), 400)
		return
	}
	fmt.Println("key: ", val)
	w.WriteHeader(http.StatusAccepted)
	err = json.NewEncoder(w).Encode(val)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
}

func PutHandler(w http.ResponseWriter, r *http.Request) {
	enableCors(&w)
	w.Header().Set("Content-Type", "application/json")
	if r.Body == nil {
		http.Error(w, "Please send a request body", 400)
		return
	}
	var message PutRequest
	fmt.Println("Deserializing request...")
	err := json.NewDecoder(r.Body).Decode(&message)
	if err != nil {
		fmt.Println(err.Error())
		http.Error(w, err.Error(), 400)
		return
	}
	ctx := context.Background()
	rdb := redisDB.GetDBClient()
	if message.Context == nil {
		message.Context = &versioning.VectorClock{}
	}
	fmt.Println("Writing to database")
	object := versioning.DataObject{
		Key:     *message.Key,
		Value:   *message.Value,
		Context: *message.Context,
	}
	marshal, err := json.Marshal(object)
	if err != nil {
		return
	}
	err = rdb.Set(ctx, *message.Key, marshal, 0).Err()
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	fmt.Println(message)
	w.WriteHeader(http.StatusAccepted)
	err = json.NewEncoder(w).Encode(message)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}
