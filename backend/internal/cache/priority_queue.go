package cache

import (
	"container/heap"
	"sync"
)

type JobQueue struct {
	Job      WarmupJob
	Priority int
	Index    int
}

type PriorityQueue struct {
	items []*JobQueue
	mu    sync.RWMutex
}

type PriorityQueueHeap []*JobQueue

func (pq PriorityQueueHeap) Len() int { return len(pq) }

func (pq PriorityQueueHeap) Less(i, j int) bool {
	return pq[i].Priority > pq[j].Priority
}

func (pq PriorityQueueHeap) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueueHeap) Push(x interface{}) {
	n := len(*pq)
	item := x.(*JobQueue)
	item.Index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueueHeap) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.Index = -1
	*pq = old[0 : n-1]
	return item
}

func NewPriorityQueue() *PriorityQueue {
	pq := &PriorityQueue{
		items: make([]*JobQueue, 0),
	}
	heap.Init((*PriorityQueueHeap)(&pq.items))
	return pq
}

func (pq *PriorityQueue) Push(job WarmupJob) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	item := &JobQueue{
		Job:      job,
		Priority: job.Priority,
	}
	heap.Push((*PriorityQueueHeap)(&pq.items), item)
}

func (pq *PriorityQueue) Pop() (WarmupJob, bool) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if len(pq.items) == 0 {
		return WarmupJob{}, false
	}

	item := heap.Pop((*PriorityQueueHeap)(&pq.items)).(*JobQueue)
	return item.Job, true
}

func (pq *PriorityQueue) Peek() (WarmupJob, bool) {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	if len(pq.items) == 0 {
		return WarmupJob{}, false
	}

	return pq.items[0].Job, true
}

func (pq *PriorityQueue) Len() int {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	return len(pq.items)
}

func (pq *PriorityQueue) Empty() bool {
	return pq.Len() == 0
}

func (pq *PriorityQueue) Clear() {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	pq.items = pq.items[:0]
	heap.Init((*PriorityQueueHeap)(&pq.items))
}

func (pq *PriorityQueue) GetJobs() []WarmupJob {
	pq.mu.RLock()
	defer pq.mu.RUnlock()

	jobs := make([]WarmupJob, len(pq.items))
	for i, item := range pq.items {
		jobs[i] = item.Job
	}
	return jobs
}
