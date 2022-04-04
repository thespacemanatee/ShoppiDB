package conHashing

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"strconv"
)
var hashRange = math.Pow(2,128) - 1

type Node struct {
	ID string
	IsSeed bool
	numTokens int

	NodeRingPositions []int
	Ring *Ring

	//{name: str, nodeRingPosition: int}
	// NodeDataArray []NodeData
}

type Ring struct {
    MaxID int // 1 to maxID inclusive
	NodesMap map[string][]int
    // RingNodeDataArray []NodeData
    // NodePrefList map[int][]NodeData //Map node/virtualNode unique hash to a list of nodeData of virtual/physical nodes belonging to another host
    // ReplicationFactor int
    // RWFactor int
    // NodeStatuses map[string]bool
}

func NewNode(nodeId string, seedNodes []string) Node {
	var isSeed = false
	for _, s := range seedNodes {
		if nodeId == s {
			isSeed = true
		}
	}
	if isSeed {
		var ring = NewRing()
		ring.NodesMap[nodeId] = []int{1}
		return Node {
			ID: nodeId,
			IsSeed: isSeed,
			Ring: ring,
		}
	} else {
		return Node {
			ID: nodeId,
			IsSeed: false,
		}
	}
}

func UpdateNode(node Node, numNodes string) {
	id, _ := strconv.Atoi(node.ID)
	totalNodes, _ := strconv.Atoi(numNodes)
	var nodeRingPositions []int
	for i := id; i <= totalNodes*totalNodes; i += totalNodes {
		nodeRingPositions = append(nodeRingPositions, i)
	}
	node.NodeRingPositions = nodeRingPositions
	// fmt.Println(node.NodeRingPositions)
}

func UpdateSeedNode(node Node, senderNodeId string) {
	node.Ring.MaxID += 1
	var nodeRingPositions []int
	id, _ := strconv.Atoi(node.ID)
	for i := id; i < node.Ring.MaxID*node.Ring.MaxID; i += node.Ring.MaxID {
		nodeRingPositions = append(nodeRingPositions, i)
	}
	// fmt.Println(nodeRingPositions)
	node.Ring.NodesMap[senderNodeId] = nodeRingPositions
}


// func (node *Node) GetSeedNodes() []string {
// 	return node.seedNodes
// }

func NewRing() *Ring {
	// nodeDataArray := make([]NodeData, maxID, maxID)
    nodesMap := make(map[string][]int)
	// fmt.Println(len(nodeDataArray))
	// fmt.Println(nodeDataArray[1].ID)
	return &Ring{MaxID:1, NodesMap: nodesMap}
}

func GetMD5Hash(text string) int64 {
	bi := big.NewInt(0)
	hasher := md5.New()
	hasher.Write([]byte(text))
	// return hex.EncodeToString(hasher.Sum(nil))
	hexstr := hex.EncodeToString(hasher.Sum(nil))
	bi.SetString(hexstr, 16)
	fmt.Println("bi:", bi)
	var hashInt = bi.Int64()
	fmt.Println("hash is:", hashInt)
	return hashInt % int64(hashRange)
	// byteArray := md5.Sum([]byte(text))
	// var output int
	// for _, num := range byteArray {
	// 	output += int(num)
	// }
	// return output % 360
}
