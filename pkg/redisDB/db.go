package redisDB

import (
	"context"
	"fmt"
	"github.com/go-redis/redis/v8"
)
type DatabaseMessage struct {
	Key string `json:"key"`
	Value string `json:"value"`
}

var rdb *redis.Client

func GetDBClient() *redis.Client {
	if rdb != nil {
		return rdb
	}
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	return rdb
}

func exampleDB() {
	ctx := context.Background()

	//Connect to the local db
	rdb := GetDBClient()

	//Set Function
	err := rdb.Set(ctx, "key", "value", 0).Err()
	if err != nil {
		panic(err)
	}

	//Get Function
	val, err := rdb.Get(ctx, "key").Result()
	if err != nil {
		panic(err)
	}
	fmt.Println("key", val)

	//Check for null
	val2, err := rdb.Get(ctx, "key2").Result()
	if err == redis.Nil {
		fmt.Println("key2 does not exist")
	} else if err != nil {
		panic(err)
	} else {
		fmt.Println("key2", val2)
	}
}
