package main

import (
	"os"
	"strings"
)

const (
	CONN_PORT = ":8080"
	CONN_TYPE = "tcp"
)

type node struct {
	membership    bool
	containerName string // for network comm
	// nodeID
	// tokenSet
	// timeOfIssue int

}

func start(done chan<- struct{}) {
	nodeMap := make(map[string]node)
}

func connect(containerName string) {

}

func getContainerName() string {
	var output string
	switch os.Getenv("NODE_ID") {
	case "0":
		output = "node0"
	case "1":
		output = "node1"
	case "2":
		output = "node3"
	}
	return output
}

func getSeedNodes() []string {
	return strings.Split(os.Getenv("SEEDNODES"), " ")
}

func nodeidToContainerName(nodeid string) string {
	var containerName string
	switch nodeid {
	case "0":
		containerName = "node0"
	case "1":
		containerName = "node1"
	case "2":
		containerName = "node2"
	}
	return containerName
}
