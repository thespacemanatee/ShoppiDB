package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	for {
		fmt.Println("NODE ID:", os.Getenv("NODE_ID"))
		fmt.Println("New")
		time.Sleep(time.Second * 10)
	}
}
