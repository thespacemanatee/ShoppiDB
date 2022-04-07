package data_versioning

import (
	"time"
)

type Clock struct {
	Counter     int64
	LastUpdated int64
}

type VectorClock map[string]Clock

type DataObject struct {
	Key     string
	Value   string
	Context VectorClock
}

// NewDataObject returns a new DataObject.
func NewDataObject(key string, rawData string) DataObject {
	return DataObject{
		Key:     key,
		Value:   rawData,
		Context: make(map[string]Clock),
	}
}

// UpdateVectorClock updates a node's clock in the object's vector clock, or generates one if this is a new object.
func UpdateVectorClock(node string, vectorClock VectorClock) {
	if clock, exists := vectorClock[node]; exists {
		clock.Counter += 1
		clock.LastUpdated = time.Now().UnixMilli()
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
	for node, _ := range a.Context {
		aIsStale = aIsStale && (a.Context[node].Counter <= b.Context[node].Counter)
	}

	// Check if b is stale
	for node, _ := range b.Context {
		bIsStale = bIsStale && (b.Context[node].Counter <= a.Context[node].Counter)
	}

	newObjects := make(map[string]DataObject)
	for k, v := range dataObjects {
		newObjects[k] = v
	}

	if aIsStale {
		newObjects[b.Key] = b
		return false, newObjects
	} else if bIsStale {
		newObjects[a.Key] = a
		return false, newObjects
	} else {
		newObjects[a.Key] = a
		newObjects[b.Key] = b
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
	conflictingObjects[dataObjects[0].Key] = dataObjects[0]
	for i := 0; i < len(dataObjects); i++ {
		for j := i + 1; j < len(dataObjects); j++ {
			_, newObjects := DeConflictDataObjects(dataObjects[i], dataObjects[j], conflictingObjects)
			conflictingObjects = newObjects
		}
	}
	return conflictingObjects
}
