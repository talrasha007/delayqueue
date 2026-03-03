package delayqueue

import (
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
