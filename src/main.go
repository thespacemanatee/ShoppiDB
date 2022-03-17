package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	for {
		time.Sleep(time.Second * 10)
		fmt.Println("NODE ID:", os.Getenv("NODE_ID"))
	}
}
