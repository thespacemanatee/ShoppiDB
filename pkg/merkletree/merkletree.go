package merkle

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"sort"

	"github.com/cbergoon/merkletree"
)

type RequestContentMap struct {
	m    map[string]*RequestContent
	tree *merkletree.MerkleTree
}

type MerkleMessage struct {
	nodeId int
	tree   *merkletree.MerkleTree
}

type RequestContent struct {
	key  string
	data string
}

//CalculateHash hashes the values of a TestContent
func (r RequestContent) CalculateHash() ([]byte, error) {
	h := sha256.New()
	if _, err := h.Write([]byte(r.data)); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

//Equals tests for equality of two Contents
func (r RequestContent) Equals(other merkletree.Content) (bool, error) {
	return r.data == other.(RequestContent).data, nil
}

func (m *RequestContentMap) sortMap() {
	keys := make([]string, 0, len(m.m))
	for k := range m.m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	tempMap := make(map[string]*RequestContent)
	for _, k := range keys {
		tempMap[k] = m.m[k]
	}
	m.m = tempMap
	m.updateTree()
}

func (m *RequestContentMap) updateTree() {
	var list []merkletree.Content
	for _, r := range m.m {
		list = append(list, r)
	}
	t, err := merkletree.NewTree(list)
	if err != nil {
		log.Fatal(err)
	}
	m.tree = t
}

func (m *RequestContentMap) compareTree(compareTree *merkletree.MerkleTree) ([]*RequestContent, error) {
	res := bytes.Compare(m.tree.MerkleRoot(), compareTree.MerkleRoot())
	result := make([]*RequestContent, 0, len(m.tree.Leafs))
	if res == 0 {
		return []*RequestContent{}, errors.New("no conflict")
	} else {
		leftResult, err := getLeafDifferences(m.tree.Root.Left.Tree, compareTree.Root.Left.Tree)
		if err != nil {
			rightResult, err := getLeafDifferences(m.tree.Root.Right.Tree, compareTree.Root.Right.Tree)
			if err != nil {
				fmt.Println("IMPOSSIBLE")
				return []*RequestContent{}, errors.New("Impossible") //IMPOSSIBLE
			}
			result = append(result, rightResult...)
			return result, nil //Result from right
		}
		result = append(result, leftResult...)
		rightResult, err := getLeafDifferences(m.tree.Root.Right.Tree, compareTree.Root.Right.Tree)
		if err != nil {
			return []*RequestContent{}, nil //Result from left
		}
		result = append(result, rightResult...)
		return result, nil //Result from both left and right
	}
}

func getLeafDifferences(m1 *merkletree.MerkleTree, m2 *merkletree.MerkleTree) ([]*RequestContent, error) {
	result := make([]*RequestContent, 0, len(m1.Leafs))
	res := bytes.Compare(m1.MerkleRoot(), m2.MerkleRoot())
	if res == 0 {
		return []*RequestContent{}, errors.New("no conflict")
	} else {
		if len(m1.Leafs) != 1 {
			leftResult, err := getLeafDifferences(m1.Root.Left.Tree, m2.Root.Left.Tree)
			if err != nil {
				rightResult, err := getLeafDifferences(m1.Root.Right.Tree, m2.Root.Right.Tree)
				if err != nil {
					fmt.Println("IMPOSSIBLE")
					return []*RequestContent{}, errors.New("Impossible") //IMPOSSIBLE
				}
				result = append(result, rightResult...)
				return result, nil //Result from right
			}
			result = append(result, leftResult...)
			rightResult, err := getLeafDifferences(m1.Root.Right.Tree, m2.Root.Right.Tree)
			if err != nil {
				return []*RequestContent{}, nil //Result from left
			}
			result = append(result, rightResult...)
			return result, nil //Result from both left and right
		} else {
			result = append(result, m2.Leafs[0].C.(*RequestContent))
			return result, nil
		}
	}
}
