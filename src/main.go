package main

import (
	"sync"
	"time"
)

type node struct {
	Membership    bool
	ContainerName string
	// nodeID
	// tokenSet
	// timeOfIssue int
}

type gossip struct {
	mu      sync.Mutex
	nodeMap map[string]node
}

func main() {
	// done := make(chan struct{})
	localNode := node{getMembership(), getLocalContainerName()}
	gossip := gossip{nodeMap: make(map[string]node)}

	//adding localNode into node map
	gossip.nodeMap[getLocalNodeID()] = localNode

	go gossip.serverStart()
	go gossip.clientStart()
	time.Sleep(time.Second * 300)
}
