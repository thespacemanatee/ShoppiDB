package node

import (
	"ShoppiDB/pkg/consistent_hashing"

	"fmt"
	"strconv"
)

type Node struct {
	nonce []string

	ID string
	IsSeed bool

	NodeRingPositions []int
	Ring *conHashing.Ring
}

func (n *Node) updateNonce(nonce string) {
	n.nonce = append(n.nonce, nonce)
	fmt.Println("Appended Nonce")
	fmt.Println(n.nonce)
}

/**
* Returns a new node
*	if is a seed node, create a new ring
*
* @param nodeId The id for the new node
* @param seedNodes The array containing id of seed nodes
*
* @return a new node
*/
func NewNode(nodeId string, seedNodes []string) Node {
	var isSeed = false
	for _, s := range seedNodes {
		if nodeId == s {
			isSeed = true
		}
	}
	if isSeed {
		var ring = conHashing.NewRing()
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

/**
* Update the position of its virtual nodes assigned
*
* @param node The node to update
* @param numNodes The total number of nodes
*
*/
func UpdateNode(node Node, numNodes string) {
	id, _ := strconv.Atoi(node.ID)
	totalNodes, _ := strconv.Atoi(numNodes)
	var nodeRingPositions []int
	for i := id; i <= totalNodes*totalNodes; i += totalNodes {
		nodeRingPositions = append(nodeRingPositions, i)
	}
	node.NodeRingPositions = nodeRingPositions
	fmt.Println(node.NodeRingPositions)
}

/**
* Update the position of its virtual nodes assigned to seed node
*
* @param node The node to update
* @param numNodes The total number of nodes
*
*/
func UpdateSeedNode(node Node, senderNodeId string) {
	node.Ring.MaxID += 1
	var nodeRingPositions []int
	id, _ := strconv.Atoi(node.ID)
	for i := id; i < node.Ring.MaxID*node.Ring.MaxID; i += node.Ring.MaxID {
		nodeRingPositions = append(nodeRingPositions, i)
	}
	fmt.Println(nodeRingPositions)
	node.Ring.NodesMap[senderNodeId] = nodeRingPositions
}