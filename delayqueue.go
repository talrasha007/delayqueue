package delayqueue

import (
	"container/heap"
	"errors"
	"sync"
	"time"
)

var ErrQueueFull = errors.New("queue is full")

type OverflowPolicy int

const (
	BlockOverflow OverflowPolicy = iota
	DropOldest
)

type Task func()

type delayedQueue struct {
	mu     sync.Mutex
	cond   *sync.Cond
	queue  taskHeap
	cap    int
	policy OverflowPolicy
	wake   chan struct{}
	seq    uint64
}

type scheduledTask struct {
	at   time.Time
	task Task
	seq  uint64
}

type taskHeap []scheduledTask

func (h taskHeap) Len() int { return len(h) }

func (h taskHeap) Less(i, j int) bool {
	if h[i].at.Equal(h[j].at) {
		return h[i].seq < h[j].seq
	}
	return h[i].at.Before(h[j].at)
}

func (h taskHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *taskHeap) Push(x any) {
	*h = append(*h, x.(scheduledTask))
}

func (h *taskHeap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	old[n-1] = scheduledTask{}
	*h = old[:n-1]
	return x
}

func New(cap int, policy OverflowPolicy) *delayedQueue {
	q := &delayedQueue{
		cap:    cap,
		policy: policy,
		wake:   make(chan struct{}, 1),
	}
	q.cond = sync.NewCond(&q.mu)
	heap.Init(&q.queue)
	go q.loop()
	return q
}

func (q *delayedQueue) Add(d time.Duration, task Task) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	st := scheduledTask{
		at:   time.Now().Add(d),
		task: task,
		seq:  q.seq,
	}
	q.seq++

	// 队列未满
	if len(q.queue) < q.cap {
		q.push(st)
		return nil
	}

	// 满载策略
	switch q.policy {
	case BlockOverflow:
		for len(q.queue) >= q.cap {
			q.cond.Wait()
		}
		q.push(st)
		return nil

	case DropOldest:
		// 删除最早未执行
		heap.Pop(&q.queue)
		q.push(st)
		q.notifyWake()
		return nil
	}

	return ErrQueueFull
}

func (q *delayedQueue) push(st scheduledTask) {
	shouldWake := len(q.queue) == 0 || st.at.Before(q.queue[0].at)
	heap.Push(&q.queue, st)
	q.cond.Signal()
	if shouldWake {
		q.notifyWake()
	}
}

func (q *delayedQueue) notifyWake() {
	select {
	case q.wake <- struct{}{}:
	default:
	}
}

func (q *delayedQueue) loop() {
	for {
		q.mu.Lock()
		for len(q.queue) == 0 {
			q.cond.Wait()
		}

		wait := time.Until(q.queue[0].at)
		if wait > 0 {
			q.mu.Unlock()
			timer := time.NewTimer(wait)
			select {
			case <-timer.C:
			case <-q.wake:
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
			}
			continue
		}

		st := heap.Pop(&q.queue).(scheduledTask)
		q.cond.Signal() // notify producers if blocked
		q.mu.Unlock()

		go st.task()
	}
}
