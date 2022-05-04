package merkle

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"

	"github.com/cbergoon/merkletree"
)

type RequestContentMap struct {
	m    map[string]RequestContent
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
	var list []merkletree.Content
	for _, k := range keys {
		list = append(list, m.m[k])
	}
	t, err := merkletree.NewTree(list)
	if err != nil {
		log.Fatal(err)
	}
	m.tree = t
}

func (m *RequestContentMap) compareTree(compareTree *merkletree.MerkleTree) ([]string, error) {
	res := bytes.Compare(m.tree.MerkleRoot(), compareTree.MerkleRoot())
	result := make([]string, 0, len(m.tree.Leafs))
	if res == 0 {
		return []string{}, errors.New("no conflict")
	} else {
		for _, c := range m.m {
			_, index, err := compareTree.GetMerklePath(c)
			if err != nil {
				log.Fatal(err)
			}
			if len(index) > 0 {
				result = append(result, c.key)
			}
		}
		return result, nil
	}
}

func (m *RequestContentMap) updateMapData(key string, data string) {
	m.m[key] = RequestContent{key: key, data: data}
	m.sortMap()
}

func main() {
	rMap := new(RequestContentMap)
	rMap.m = make(map[string]RequestContent)
	rMap1 := new(RequestContentMap)
	rMap1.m = make(map[string]RequestContent)

	for i := 0; i < 3; i++ {
		//key := strconv.Itoa(i + rand.Intn(100))
		key := strconv.Itoa(i)
		data := key + "is data"
		rMap.updateMapData(key, data)
	}
	for i := 0; i < 3; i++ {
		//key := strconv.Itoa(i + rand.Intn(100))
		key := strconv.Itoa(i)
		data := key + "is data2"
		if i < 1 {
			data = key + "is data"
		}
		rMap1.updateMapData(key, data)
	}
	fmt.Println(rMap)
	fmt.Println(rMap1)
	fmt.Println(rMap.tree.MerkleRoot())
	fmt.Println(rMap1.tree.MerkleRoot())
	for _, c := range rMap.m {
		fmt.Println(c)
		path, index, err := rMap.tree.GetMerklePath(c)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Path: ", path)
		fmt.Println("Index: ", index)
	}
	for _, c := range rMap1.m {
		fmt.Println(c)
		path, index, err := rMap.tree.GetMerklePath(c)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Path: ", path)
		fmt.Println("Index: ", index)
	}
}
