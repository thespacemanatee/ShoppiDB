package conHashing

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math/big"
)

type Node struct{
	ID string
	IsSeed bool

	NodeRingPositions []int
	Ring *Ring

	//{name: str, nodeRingPosition: int}
	// NodeDataArray []NodeData
}

type Ring struct{
    MaxID int // 0 to maxID inclusive
    // RingNodeDataArray []NodeData
    // NodePrefList map[int][]NodeData //Map node/virtualNode unique hash to a list of nodeData of virtual/physical nodes belonging to another host
    // ReplicationFactor int
    // RWFactor int
    // NodeStatuses map[string]bool
}

func NewNode(nodeId string, maxNodes int, seedNodes []string) Node{
	var isSeed = false
	for _, s := range seedNodes {
		if nodeId == s {
			isSeed = true
		}
	}
	if isSeed {
		fmt.Println("is seed")
		var ring = NewRing(maxNodes)
		return Node{
			ID: nodeId,
			IsSeed: isSeed,
			Ring: ring,
		}
	} else {
		fmt.Println("not seed")
		return Node{
			ID: nodeId,
			IsSeed: false,
		}
	}
}

// func (node *Node) GetSeedNodes() []string {
// 	return node.seedNodes
// }

func NewRing(maxNodes int) *Ring {
	// nodeDataArray := make([]NodeData, maxID, maxID)
    // nodePrefList := make(map[int][]NodeData, maxID)
	// fmt.Println(len(nodeDataArray))
	// fmt.Println(nodeDataArray[1].ID)
	return &Ring{MaxID:maxNodes}
}

func GetMD5Hash(text string) int64 {
	bi := big.NewInt(0)
	hasher := md5.New()
	hasher.Write([]byte(text))
	// return hex.EncodeToString(hasher.Sum(nil))
	hexstr := hex.EncodeToString(hasher.Sum(nil))
	bi.SetString(hexstr, 16)
	var hashInt = bi.Int64()
	fmt.Println("hash is:", hashInt%360)
	return hashInt % 360
	// byteArray := md5.Sum([]byte(text))
	// var output int
	// for _, num := range byteArray {
	// 	output += int(num)
	// }
	// return output % 360
}
