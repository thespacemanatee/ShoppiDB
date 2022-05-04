package data_versioning

import (
	"reflect"
	"testing"
	"time"
)

func TestDeConflictVectorClocks(t *testing.T) {
	m1 := make(VectorClock)
	m1["A"] = Clock{
		Counter:     1,
		LastUpdated: time.Now().UnixMilli(),
	}
	m2 := make(VectorClock)
	m2["B"] = Clock{
		Counter:     1,
		LastUpdated: time.Now().UnixMilli(),
	}
	m3 := make(VectorClock)
	m3["C"] = Clock{
		Counter:     1,
		LastUpdated: time.Now().UnixMilli(),
	}
	m4 := make(VectorClock)
	m4["A"] = Clock{
		Counter:     1,
		LastUpdated: time.Now().UnixMilli(),
	}
	m4["B"] = Clock{
		Counter:     1,
		LastUpdated: time.Now().UnixMilli(),
	}
	m4["C"] = Clock{
		Counter:     1,
		LastUpdated: time.Now().UnixMilli(),
	}
	m5 := make(VectorClock)
	m5["A"] = Clock{
		Counter:     1,
		LastUpdated: time.Now().UnixMilli(),
	}
	m5["C"] = Clock{
		Counter:     1,
		LastUpdated: time.Now().UnixMilli(),
	}
	m5["D"] = Clock{
		Counter:     1,
		LastUpdated: time.Now().UnixMilli(),
	}
	m6 := make(VectorClock)
	m6["B"] = Clock{
		Counter:     1,
		LastUpdated: time.Now().UnixMilli(),
	}
	m6["C"] = Clock{
		Counter:     1,
		LastUpdated: time.Now().UnixMilli(),
	}
	m6["A"] = Clock{
		Counter:     1,
		LastUpdated: time.Now().UnixMilli(),
	}

	//Slice of test cases
	tests := []struct {
		name            string
		VectorClocks    []VectorClock
		ExpectedOutcome bool
	}{
		{"Single Vector Clock", []VectorClock{m1, m1}, false},
		{"Different Single Vector Clock", []VectorClock{m1, m2}, true},
		{"Multiple Vector Clock", []VectorClock{m4, m4}, false},
		{"Different Multiple Vector Clock", []VectorClock{m4, m5}, true},
	}

	res := make(map[string]DataObject)
	for _, testCase := range tests {
		conflicted, _ := DeConflictDataObjects(DataObject{
			Key:     "cart-1337",
			Value:   "test",
			Context: testCase.VectorClocks[0],
		}, DataObject{
			Key:     "cart-1337",
			Value:   "test",
			Context: testCase.VectorClocks[1],
		}, res)
		if reflect.DeepEqual(conflicted, testCase.ExpectedOutcome) != true {
			t.Errorf("Expected %v;\n Got %v\n", testCase.ExpectedOutcome, conflicted)
		}
	}
}
