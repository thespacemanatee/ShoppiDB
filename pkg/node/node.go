package node

import (
	"fmt"
)

type Node struct {
	nonce         []string
	ContainerName string
	TokenSet      [][]int
}

func (n *Node) updateNonce(nonce string) {
	n.nonce = append(n.nonce, nonce)
	fmt.Println("Appended Nonce")
	fmt.Println(n.nonce)
}
