package delayqueue

import (
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
	mu      sync.Mutex
	cond    *sync.Cond
	queue   []scheduledTask
	cap     int
	policy  OverflowPolicy
	running bool
}

type scheduledTask struct {
	at   time.Time
	task Task
}

func New(cap int, policy OverflowPolicy) *delayedQueue {
	q := &delayedQueue{
		cap:    cap,
		policy: policy,
	}
	q.cond = sync.NewCond(&q.mu)
	go q.loop()
	return q
}

func (q *delayedQueue) Add(d time.Duration, task Task) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// 队列未满
	if len(q.queue) < q.cap {
		q.push(scheduledTask{at: time.Now().Add(d), task: task})
		q.cond.Signal()
		return nil
	}

	// 满载策略
	switch q.policy {
	case BlockOverflow:
		for len(q.queue) >= q.cap {
			q.cond.Wait()
		}
		q.push(scheduledTask{at: time.Now().Add(d), task: task})
		q.cond.Signal()
		return nil

	case DropOldest:
		// 删除最早未执行
		q.queue[0] = scheduledTask{}
		q.queue = q.queue[1:]
		q.push(scheduledTask{at: time.Now().Add(d), task: task})
		q.cond.Signal()
		return nil
	}

	return ErrQueueFull
}

func (q *delayedQueue) push(st scheduledTask) {
	q.queue = append(q.queue, st)
}

func (q *delayedQueue) loop() {
	for {
		q.mu.Lock()
		for len(q.queue) == 0 {
			q.cond.Wait()
		}

		now := time.Now()
		// 找最先要执行的任务
		minIdx := -1
		var mint time.Time
		for i, t := range q.queue {
			if minIdx == -1 || t.at.Before(mint) {
				minIdx = i
				mint = t.at
			}
		}

		wait := mint.Sub(now)
		if wait > 0 {
			q.mu.Unlock()
			time.Sleep(wait)
			continue
		}

		st := q.queue[minIdx]
		// 从队列删除
		q.queue = append(q.queue[:minIdx], q.queue[minIdx+1:]...)
		q.cond.Signal() // notify producers if blocked
		q.mu.Unlock()

		go st.task()
	}
}
