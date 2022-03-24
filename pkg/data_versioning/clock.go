package data_versioning

import (
	"encoding/json"
	"time"
)

type Node string

type Counter int

type Clock struct {
	Counter     Counter
	LastUpdated int64
}

type VectorClock map[Node]Clock

type DataObject struct {
	ObjectId string
	RawData  json.RawMessage
	Version  VectorClock
}

// NewDataObject returns a new DataObject.
func NewDataObject(id string, rawData json.RawMessage) DataObject {
	return DataObject{
		ObjectId: id,
		RawData:  rawData,
		Version:  make(map[Node]Clock),
	}
}

// UpdateVectorClock updates a node's clock in the object's vector clock, or generates one if this is a new object.
func UpdateVectorClock(node Node, vectorClock VectorClock) {
	if clock, exists := vectorClock[node]; exists {
		clock.Counter += 1
		vectorClock[node] = clock
	} else {
		vectorClock[node] = Clock{Counter: 1, LastUpdated: time.Now().UnixMilli()}
	}
}

// DeConflictDataObjects returns true if there is an unresolvable conflict, and a new set of data objects after trying syntatic reconciliation.
func DeConflictDataObjects(a DataObject, b DataObject, dataObjects map[string]DataObject) (bool, map[string]DataObject) {
	aIsStale := true
	bIsStale := true

	// Check if a is stale
	for node, _ := range a.Version {
		aIsStale = aIsStale && (a.Version[node].Counter <= b.Version[node].Counter)
	}

	// Check if b is stale
	for node, _ := range b.Version {
		bIsStale = bIsStale && (b.Version[node].Counter <= a.Version[node].Counter)
	}

	newObjects := make(map[string]DataObject)
	for k, v := range dataObjects {
		newObjects[k] = v
	}

	if aIsStale {
		newObjects[b.ObjectId] = b
		return false, newObjects
	} else if bIsStale {
		newObjects[a.ObjectId] = a
		return false, newObjects
	} else {
		newObjects[a.ObjectId] = a
		newObjects[b.ObjectId] = b
		return true, newObjects
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
			_, newObjects := DeConflictDataObjects(dataObjects[i], dataObjects[j], conflictingObjects)
			conflictingObjects = newObjects
		}
	}
	return conflictingObjects
}
