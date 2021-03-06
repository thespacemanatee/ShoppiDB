package consistent_hashing

import (
	"crypto/md5"
	"encoding/hex"
	"math/big"
	"sync"
)
var hashRange = new(big.Int).Exp(big.NewInt(2), big.NewInt(128), nil)

// struct to prevent duplicate
type VirtualPositions struct {
	mu sync.Mutex
	AssignedRange []int
}

func (v *VirtualPositions) GetVirtualPositions() []int {
	v.mu.Lock()
    defer v.mu.Unlock()
	return v.AssignedRange
}

func (v *VirtualPositions) UpdateVirtualPositions(input int) {
	v.mu.Lock()
	v.AssignedRange = append(v.AssignedRange, input)
	v.mu.Unlock()
}

/**
* Returns the position indicated by the hash
* Eg. return value = 0.8
* position = 0.8 * 64
* 
* @param text The text to hash
* 
* @return the float generated by the hash
*/
func GetMD5Hash(text string) *big.Float {
	bi := big.NewInt(0)
	hasher := md5.New()
	hasher.Write([]byte(text))
	hexstr := hex.EncodeToString(hasher.Sum(nil))
	bi.SetString(hexstr, 16)
	hashFloat := new(big.Float).SetInt(bi)
	hashRangeFloat := new(big.Float).SetInt(hashRange)
	hash := new(big.Float).Quo(hashFloat,hashRangeFloat)
	return new(big.Float).Mul(hash, big.NewFloat(64))
}
