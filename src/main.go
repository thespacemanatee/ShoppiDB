package main

import (
	"fmt"
	"os"
	s "strings"
	"time"
)

func main() {
	// done := make(chan struct{})

	for {
		seedNodes := os.Getenv("SEEDNODES")
		seednodes := s.Split(seedNodes, " ")
		containerName := getContainerName()
		fmt.Println("NODE ID:", os.Getenv("NODE_ID"))
		fmt.Println("Membership is", os.Getenv("MEMBERSHIP"))
		fmt.Println("seedNodes =", seednodes[0], seednodes[1])
		fmt.Println("my container name is", containerName)
		fmt.Println("New")
		time.Sleep(time.Second * 10)
	}
}
