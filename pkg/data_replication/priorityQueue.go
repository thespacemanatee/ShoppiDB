package replication


type Item struct {
	req ClientReq
	Index   int // The index of the item in the heap.
}

type PriorityQueue []*Item

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the lowest based on expiration number as the priority
	// The lower the expiry, the higher the priority
	if pq[i].req.Timestamp.Before(pq[j].req.Timestamp) {
		return true
	} else {
		return false
	}
}

// We just implement the pre-defined function in interface of heap.

// returns value at head of queue
func (pq *PriorityQueue) Front() ClientReq {
	n := len(*pq)
	if n > 0 {
		q := *pq
		item := q[0]
		return item.req
	} else {
		return ClientReq{}
	}
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.Index = -1
	*pq = old[0 : n-1]
	return item
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*Item)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}
