package core

import "sync"

// Item is an interface for objects that can be compared for priority ordering
type Item interface {
	Less(Item) bool
}

// PriorityQueue implements a thread-safe min-heap priority queue
// Lower priority items (as determined by Less) are dequeued first
type PriorityQueue struct {
	sync.Mutex
	length          int
	data            []Item
	notifyCallbacks []func(Item)
}

// NewPriorityQueue creates a new priority queue with the provided items
// The items will be heapified during initialization
func NewPriorityQueue(data []Item) *PriorityQueue {
	q := &PriorityQueue{
		data:   data,
		length: len(data),
	}

	// Heapify if there's initial data
	if q.length > 0 {
		for i := q.length >> 1; i >= 0; i-- {
			q.down(i)
		}
	}

	return q
}

// Push adds an item to the priority queue
// Thread-safe operation
func (q *PriorityQueue) Push(item Item) {
	q.Lock()
	defer q.Unlock()

	q.data = append(q.data, item)
	q.length++
	q.up(q.length - 1)

	// Notify any subscribers of new items
	for _, notify := range q.notifyCallbacks {
		go notify(item)
	}
}

// PopLock returns a channel that will receive the next item
// when one becomes available
func (q *PriorityQueue) PopLock() <-chan Item {
	ch := make(chan Item)
	q.notifyCallbacks = append(q.notifyCallbacks, func(_ Item) {
		ch <- q.Pop()
	})
	return ch
}

// Pop removes and returns the highest priority (lowest) item
// Thread-safe operation
func (q *PriorityQueue) Pop() Item {
	q.Lock()
	defer q.Unlock()

	if q.length == 0 {
		return nil
	}

	top := q.data[0]
	q.length--

	if q.length > 0 {
		// Move the last item to the top and restore heap property
		q.data[0] = q.data[q.length]
		q.down(0)
	}

	// Reduce slice capacity to prevent memory leaks
	q.data = q.data[:q.length]

	return top
}

// Peek returns the highest priority item without removing it
// Thread-safe operation
func (q *PriorityQueue) Peek() Item {
	q.Lock()
	defer q.Unlock()

	if q.length == 0 {
		return nil
	}
	return q.data[0]
}

// Len returns the number of items in the queue
// Thread-safe operation
func (q *PriorityQueue) Len() int {
	q.Lock()
	defer q.Unlock()

	return q.length
}

// down moves an item down the heap until the heap property is restored
// Part of the heap implementation (not exported)
func (q *PriorityQueue) down(pos int) {
	data := q.data
	halfLength := q.length >> 1
	item := data[pos]

	for pos < halfLength {
		// Calculate indices of children
		left := (pos << 1) + 1
		right := left + 1

		// Find the child with lower priority (minimum)
		best := data[left]
		bestPos := left

		if right < q.length && data[right].Less(best) {
			bestPos = right
			best = data[right]
		}

		// If heap property is satisfied, stop
		if !best.Less(item) {
			break
		}

		// Move the child up
		data[pos] = best
		pos = bestPos
	}

	// Place the item in its final position
	data[pos] = item
}

// up moves an item up the heap until the heap property is restored
// Part of the heap implementation (not exported)
func (q *PriorityQueue) up(pos int) {
	data := q.data
	item := data[pos]

	for pos > 0 {
		parent := (pos - 1) >> 1
		current := data[parent]

		// If heap property is satisfied, stop
		if !item.Less(current) {
			break
		}

		// Move parent down
		data[pos] = current
		pos = parent
	}

	// Place the item in its final position
	data[pos] = item
}
