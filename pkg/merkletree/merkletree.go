package merkle

import (
	"ShoppiDB/pkg/httpClient"
	"ShoppiDB/pkg/redisDB"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"sync"

	"github.com/cbergoon/merkletree"
)

type RequestContentMap struct {
	M    map[string]RequestContent
	Tree *merkletree.MerkleTree
}

type MerkleMessage struct {
	NodeName string                 `json:"nodename"`
	Tree     *merkletree.MerkleTree `json:"tree"`
	HashNo   int                    `json:"hashno"`
}

type MerkleUpdateMessage struct {
	Keys []string `json:"keys"`
}

type MerkleReplyUpdateMessage struct {
	Result map[string]string `json:"result"`
}

type MerkleInitateMessage struct {
	NodeName    string `json:"nodename"`
	HashNumbers []int  `json:"hashnumbers"`
}

type RequestContent struct {
	key  string
	data string
}

type dataObject struct {
	Key   string
	Value string
}

type Merkler struct {
	HashNumberToKey map[int]*RequestContentMap
	mu              *sync.Mutex
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
	keys := make([]string, 0, len(m.M))
	for k := range m.M {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var list []merkletree.Content
	for _, k := range keys {
		list = append(list, m.M[k])
	}
	t, err := merkletree.NewTree(list)
	if err != nil {
		log.Fatal(err)
	}
	m.Tree = t
}

func (m *RequestContentMap) CompareTree(compareTree *merkletree.MerkleTree) ([]string, error) {
	res := bytes.Compare(m.Tree.MerkleRoot(), compareTree.MerkleRoot())
	result := make([]string, 0, len(m.Tree.Leafs))
	if res == 0 {
		return []string{}, errors.New("no conflict")
	} else {
		for _, c := range m.M {
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

func (m *RequestContentMap) updateMapDataContent(key string, data string) {
	m.M[key] = RequestContent{key: key, data: data}
	m.sortMap()
}

func (merkler *Merkler) UpdateMapData(hashNo int, key string, data string) {
	merkler.mu.Lock()
	merkler.HashNumberToKey[hashNo].updateMapDataContent(key, data)
	merkler.mu.Unlock()
}

func (merkler *Merkler) ReceivedMerkleTree(hashNo int, compareTree *merkletree.MerkleTree, target string) {
	merkler.mu.Lock()
	result, err := merkler.HashNumberToKey[hashNo].CompareTree(compareTree)
	merkler.mu.Unlock()
	if err != nil {
		fmt.Println("Merkle tree has no conflict")
	} else { //Ask for updated data based on key
		defer func(hashNo int, compareTree *merkletree.MerkleTree, target string) {
			if r := recover(); r != nil {
				fmt.Println("Panic occur, process recovered", r)
				go merkler.ReceivedMerkleTree(hashNo, compareTree, target)
			}
		}(hashNo, compareTree, target)
		msg := MerkleUpdateMessage{Keys: result}
		client := httpClient.GetHTTPClient()
		msgJson, err1 := json.Marshal(msg)
		checkErr(err1)
		req, err2 := http.NewRequest(http.MethodPost, target, bytes.NewBuffer(msgJson))
		checkErr(err2)
		req.Header.Set("Content-Type", "application/json")
		resp, err3 := client.Do(req)
		checkErr(err3)
		var respMsg MerkleReplyUpdateMessage
		err := json.NewDecoder(resp.Body).Decode(&respMsg)
		checkErr(err)
		fmt.Println("Successfully ask for merkle update", respMsg.Result)
		merkler.mu.Lock()
		ctx := context.Background()
		rdb := redisDB.GetDBClient()
		for key, val := range respMsg.Result {
			merkler.HashNumberToKey[hashNo].updateMapDataContent(key, val)
			newObject := dataObject{Key: key, Value: val}
			marshal, err := json.Marshal(newObject)
			checkErr(err)
			fmt.Println("Writing to database")
			err = rdb.Set(ctx, key, marshal, 0).Err()
			checkErr(err)
			fmt.Println("Successfully write to db and merkle tree")
		}
		merkler.mu.Unlock()
	}
}

func (merkler *Merkler) InitiateMerkleCheck(hashNos []int, targetNode string) {
	failList := make([]int, 0, len(hashNos))
	for _, hashNo := range hashNos {
		defer func(hashNo int) {
			if r := recover(); r != nil {
				fmt.Println("Panic occur, process recovered", r)
				failList = append(failList, hashNo)
			}
		}(hashNo)
		target := "http://" + targetNode + ":8080" + "/merklecheck"
		merkler.mu.Lock()
		msg := MerkleMessage{NodeName: targetNode, Tree: merkler.HashNumberToKey[hashNo].Tree, HashNo: hashNo}
		merkler.mu.Unlock()
		client := httpClient.GetHTTPClient()
		msgJson, err1 := json.Marshal(msg)
		checkErr(err1)
		req, err2 := http.NewRequest(http.MethodPost, target, bytes.NewBuffer(msgJson))
		checkErr(err2)
		req.Header.Set("Content-Type", "application/json")
		resp, err3 := client.Do(req)
		checkErr(err3)
		var respMsg string
		err := json.NewDecoder(resp.Body).Decode(&respMsg)
		checkErr(err)
		fmt.Println("Successful initiate the check for hashno", hashNo)
	}
	if len(failList) > 0 {
		go merkler.InitiateMerkleCheck(failList, targetNode)
	}
}

func checkErr(err error) {
	if err != nil {
		fmt.Println(err)
		return
	}
}

func getLocalContainerName() (output string) {
	output = "node" + os.Getenv("NODE_ID")
	return output
}

// func main() {
// 	rMap := new(RequestContentMap)
// 	rMap.M = make(map[string]RequestContent)
// 	rMap1 := new(RequestContentMap)
// 	rMap1.M = make(map[string]RequestContent)

// 	for i := 0; i < 3; i++ {
// 		//key := strconv.Itoa(i + rand.Intn(100))
// 		key := strconv.Itoa(i)
// 		data := key + "is data"
// 		rMap.updateMapData(key, data)
// 	}
// 	for i := 0; i < 3; i++ {
// 		//key := strconv.Itoa(i + rand.Intn(100))
// 		key := strconv.Itoa(i)
// 		data := key + "is data2"
// 		if i < 1 {
// 			data = key + "is data"
// 		}
// 		rMap1.updateMapData(key, data)
// 	}
// 	fmt.Println(rMap)
// 	fmt.Println(rMap1)
// 	fmt.Println(rMap.Tree.MerkleRoot())
// 	fmt.Println(rMap1.Tree.MerkleRoot())
// 	for _, c := range rMap.M {
// 		fmt.Println(c)
// 		path, index, err := rMap.Tree.GetMerklePath(c)
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		fmt.Println("Path: ", path)
// 		fmt.Println("Index: ", index)
// 	}
// 	for _, c := range rMap1.M {
// 		fmt.Println(c)
// 		path, index, err := rMap.Tree.GetMerklePath(c)
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		fmt.Println("Path: ", path)
// 		fmt.Println("Index: ", index)
// 	}
// }
