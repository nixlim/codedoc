package todolist

import (
	"container/heap"
	"fmt"
)

// PriorityQueue implements a priority queue for TODO items.
// Items with higher priority values are processed first.
type PriorityQueue struct {
	items    []TodoItem
	itemMap  map[string]*TodoItem
	progress Progress
}

// NewPriorityQueue creates a new priority queue.
func NewPriorityQueue() *PriorityQueue {
	pq := &PriorityQueue{
		items:   make([]TodoItem, 0),
		itemMap: make(map[string]*TodoItem),
		progress: Progress{
			Total:      0,
			Pending:    0,
			InProgress: 0,
			Complete:   0,
			Failed:     0,
			Skipped:    0,
		},
	}
	heap.Init(pq)
	return pq
}

// Len returns the number of items in the queue.
func (pq *PriorityQueue) Len() int {
	return len(pq.items)
}

// Less compares two items for ordering.
// Higher priority items come first.
func (pq *PriorityQueue) Less(i, j int) bool {
	return pq.items[i].Priority > pq.items[j].Priority
}

// Swap exchanges two items in the queue.
func (pq *PriorityQueue) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
}

// Push adds an item to the queue.
func (pq *PriorityQueue) Push(x interface{}) {
	item := x.(TodoItem)
	pq.items = append(pq.items, item)
	pq.itemMap[item.FilePath] = &pq.items[len(pq.items)-1]

	// Update progress
	pq.progress.Total++
	pq.updateStatusCount(item.Status, 1)
}

// Pop removes and returns the highest priority item.
func (pq *PriorityQueue) Pop() interface{} {
	n := len(pq.items)
	if n == 0 {
		return nil
	}

	item := pq.items[n-1]
	pq.items = pq.items[0 : n-1]
	delete(pq.itemMap, item.FilePath)

	// Update progress
	pq.updateStatusCount(item.Status, -1)

	return item
}

// AddItem adds a new TODO item to the queue.
func (pq *PriorityQueue) AddItem(item TodoItem) {
	heap.Push(pq, item)
}

// PopNext retrieves and removes the next pending item from the queue.
func (pq *PriorityQueue) PopNext() (*TodoItem, error) {
	// Find the highest priority pending item
	var bestItem *TodoItem
	bestIdx := -1

	for i, item := range pq.items {
		if item.Status == ItemStatusPending {
			if bestItem == nil || item.Priority > bestItem.Priority {
				bestItem = &pq.items[i]
				bestIdx = i
			}
		}
	}

	if bestItem == nil {
		return nil, fmt.Errorf("no pending items in queue")
	}

	// Mark as in progress
	bestItem.Status = ItemStatusInProgress
	pq.updateStatusCount(ItemStatusPending, -1)
	pq.updateStatusCount(ItemStatusInProgress, 1)

	// Remove from queue
	heap.Remove(pq, bestIdx)

	return bestItem, nil
}

// UpdateStatus updates the status of an item.
func (pq *PriorityQueue) UpdateStatus(filePath string, status ItemStatus) error {
	item, exists := pq.itemMap[filePath]
	if !exists {
		// Item might have been popped, check if we're updating a processed item
		return nil
	}

	oldStatus := item.Status
	item.Status = status

	// Update progress counts
	pq.updateStatusCount(oldStatus, -1)
	pq.updateStatusCount(status, 1)

	return nil
}

// GetProgress returns the current progress statistics.
func (pq *PriorityQueue) GetProgress() *Progress {
	// Return a copy to prevent external modification
	progress := pq.progress
	return &progress
}

// updateStatusCount updates the progress count for a status.
func (pq *PriorityQueue) updateStatusCount(status ItemStatus, delta int) {
	switch status {
	case ItemStatusPending:
		pq.progress.Pending += delta
	case ItemStatusInProgress:
		pq.progress.InProgress += delta
	case ItemStatusComplete:
		pq.progress.Complete += delta
	case ItemStatusFailed:
		pq.progress.Failed += delta
	case ItemStatusSkipped:
		pq.progress.Skipped += delta
	}
}

// Clear removes all items from the queue.
func (pq *PriorityQueue) Clear() {
	pq.items = make([]TodoItem, 0)
	pq.itemMap = make(map[string]*TodoItem)
	pq.progress = Progress{
		Total:      0,
		Pending:    0,
		InProgress: 0,
		Complete:   0,
		Failed:     0,
		Skipped:    0,
	}
	heap.Init(pq)
}
