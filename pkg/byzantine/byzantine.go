package byzantine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

type ByzantineMessage struct {
	Read  bool   `json:"read"`
	Dest  int    `json:"dest"`
	Key   string `json:"key"`
	Value string `json:"value"`
	Nonce string `json:"nonce"`
}

// func sendByzantineRequest(input string) (string, error) {
// 	switch input {
// 	default:
// 		return "", error.New("Invalid input")
// 	case "write":
// 		sendByzantineWrite()
// 		return "Proceed to write", nil
// 	case "reply":
// 		sendByzantineRead()
// 		return "Proceed to read", nil
// 	}
// }

func SendByzantineInitiateRead(httpClient *http.Client, client int, nodeId []int, key string) {
	for _, i := range nodeId {
		go sendByzantineReadRequest(httpClient, client, i, key)
		time.Sleep(time.Millisecond * 10)
	}
}

func sendByzantineReadRequest(httpClient *http.Client, client int, nodeId int, keyString string) {
	fmt.Println("Sending read request to replica nodes")
	target := "http://node" + strconv.Itoa(nodeId) + ":8080/byzantine"
	nonce := generateNonce()
	//updateNonce(nonce)
	msg := ByzantineMessage{Read: true, Dest: client, Key: keyString, Nonce: nonce}
	byzantineMessageJson, err := json.Marshal(msg)
	req, err := http.NewRequest(http.MethodPost, target, bytes.NewBuffer(byzantineMessageJson))
	checkErr(err)
	req.Header.Set("Content-Type", "application/json")
	fmt.Println(req)
	resp, err := httpClient.Do(req)
	checkErr(err)
	defer resp.Body.Close()
	var receivedMessage string
	json.NewDecoder(resp.Body).Decode(&receivedMessage)
	fmt.Println(receivedMessage)

}

func generateNonce() string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, 8)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}

func checkErr(err error) {

	if err != nil {
		fmt.Println(err)
	}
}
