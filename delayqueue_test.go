package delayqueue

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDelayqueueWait(t *testing.T) {
	q := New(2, BlockOverflow)

	q.Add(3*time.Second, func() {
		println("task 3 executed")
	})
	q.Add(2*time.Second, func() {
		println("task 2 executed")
	})
	q.Add(1*time.Second, func() {
		println("task 1 executed")
	})

	time.Sleep(4 * time.Second)
}

func TestDelayqueueDrop(t *testing.T) {
	q := New(2, DropOldest)

	q.Add(3*time.Second, func() {
		println("task 3 executed")
	})
	q.Add(2*time.Second, func() {
		println("task 2 executed")
	})
	q.Add(1*time.Second, func() {
		println("task 1 executed")
	})

	time.Sleep(4 * time.Second)
}

func TestDelayqueueBlockOverflowStress(t *testing.T) {
	const (
		capacity    = 8
		producers   = 16
		perProducer = 25
		total       = producers * perProducer
	)

	q := New(capacity, BlockOverflow)

	var executed atomic.Int64
	seen := make([]atomic.Int32, total)
	start := make(chan struct{})
	errs := make(chan error, total)

	var addWG sync.WaitGroup
	var taskWG sync.WaitGroup
	taskWG.Add(total)

	for p := 0; p < producers; p++ {
		p := p
		addWG.Add(1)
		go func() {
			defer addWG.Done()
			<-start

			for i := 0; i < perProducer; i++ {
				id := p*perProducer + i
				err := q.Add(20*time.Millisecond, func() {
					seen[id].Add(1)
					executed.Add(1)
					taskWG.Done()
				})
				if err != nil {
					errs <- err
					return
				}

				if size := q.Size(); size > capacity {
					errs <- ErrQueueFull
					return
				}
			}
		}()
	}

	close(start)

	if !waitTimeout(&addWG, 3*time.Second) {
		t.Fatal("Add blocked too long under BlockOverflow stress")
	}
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("Add returned unexpected error: %v", err)
		}
	}

	if !waitTimeout(&taskWG, 3*time.Second) {
		t.Fatalf("timed out waiting for tasks, executed=%d want=%d", executed.Load(), total)
	}

	if got := executed.Load(); got != total {
		t.Fatalf("executed task count = %d, want %d", got, total)
	}

	for id := range seen {
		if got := seen[id].Load(); got != 1 {
			t.Fatalf("task %d executed %d times, want 1", id, got)
		}
	}
}

func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		defer close(done)
		wg.Wait()
	}()

	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}
