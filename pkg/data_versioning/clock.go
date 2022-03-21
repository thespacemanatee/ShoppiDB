package data_versioning

import "time"

type Node string

type Counter int

type Clock struct {
	counter     Counter
	lastUpdated time.Time
}

type VectorClock map[Node]Clock

// UpdateVectorClock updates a node's clock in the object's vector clock
func UpdateVectorClock(node Node, vectorClock VectorClock) {
	if clock, exists := vectorClock[node]; exists {
		clock.counter += 1
		vectorClock[node] = clock
	} else {
		vectorClock[node] = Clock{counter: 1, lastUpdated: time.Now()}
	}
}

func IsConflictingVectorClocks(a VectorClock, b VectorClock) bool {
	if !equalNodes(a, b) {

	}
}

// GetConflictingVectorClocks returns an array of conflicting vector clocks
func GetConflictingVectorClocks(vectorClocks []VectorClock) []VectorClock {
	var conflictingClocks []VectorClock
	for i := 0; i < len(vectorClocks); i++ {
		conflict := true
		for j := 0; j < len(vectorClocks); j++ {
			if i != j {
				if !IsConflictingVectorClocks(vectorClocks[i], vectorClocks[j]) {
					conflict = false
				}
			}
		}
		if conflict {
			conflictingClocks = append(conflictingClocks, vectorClocks[i])
		}
	}
	return conflictingClocks
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
