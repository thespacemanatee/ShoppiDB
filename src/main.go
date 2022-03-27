package main

import (
	gossip "ShoppiDB/src/gossipProtocol"
	"time"
)

func main() {
	// done := make(chan struct{})
	localNode := gossip.Node{Membership: gossip.GetMembership(), ContainerName: gossip.GetLocalContainerName()}
	toCommunicate := gossip.Gossip{NodeMap: make(map[string]gossip.Node)}

	//adding localNode into node map
	toCommunicate.NodeMap[gossip.GetLocalNodeID()] = localNode

	go toCommunicate.ServerStart()
	go toCommunicate.ClientStart()
	time.Sleep(time.Minute * 5)
}
