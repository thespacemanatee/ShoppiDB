package data_versioning

import (
	"encoding/json"
	"time"
)

type Node string

type Counter int

type Clock struct {
	Counter     Counter
	LastUpdated time.Time
}

type VectorClock map[Node]Clock

type DataObject struct {
	ObjectId string
	RawData  json.RawMessage
	Version  VectorClock
}

// UpdateVectorClock updates a node's clock in the object's vector clock
func UpdateVectorClock(node Node, vectorClock VectorClock) {
	if clock, exists := vectorClock[node]; exists {
		clock.Counter += 1
		vectorClock[node] = clock
	} else {
		vectorClock[node] = Clock{Counter: 1, LastUpdated: time.Now()}
	}
}

func CompareVectorClocks(a VectorClock, b VectorClock, dataObjects map[string]DataObject) bool {
	if !equalNodes(a, b) {
		// Case 1: a and b are the same length
		// 		if a is strictly less than b, return b only
		// 		if a is not strictly smaller than b, return both
		// Case 2: a is shorter than b
		// 		if a is not strictly smaller than b, return both
		// 		if a is strictly less than b, return b only
		// Longer is just vice versa
	}
}

// GetResponseDataObjects returns an array of data objects to be returned to the client. If there are more than
// one objects, semantic reconciliation is required at the client.
func GetResponseDataObjects(dataObjects []DataObject) map[string]DataObject {
	if len(dataObjects) == 0 {
		return map[string]DataObject{}
	}
	// Create a set that always has at least 1 data object (first object in array)
	conflictingObjects := make(map[string]DataObject)
	conflictingObjects[dataObjects[0].ObjectId] = dataObjects[0]
	for i := 0; i < len(dataObjects); i++ {
		for j := i + 1; j < len(dataObjects); j++ {
			CompareVectorClocks(dataObjects[i].Version, dataObjects[j].Version, conflictingObjects)
		}
	}
	return conflictingObjects
}

// Returns true if vector a and b have the same nodes.
func equalNodes(a, b VectorClock) bool {
	if len(a) != len(b) {
		return false
	}
	for node, _ := range a {
		_, exists := b[node]
		if exists == false {
			return false
		}
	}
	return true
}
