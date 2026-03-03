# delayqueue

A lightweight in-memory delayed task queue for Go.

This library lets you schedule functions to run after a given delay and choose what happens when the queue is full.

## Install

```bash
go get github.com/talrasha007/delayqueue
```

## Quick Start

```go
package main

import (
	"time"

	"github.com/talrasha007/delayqueue"
)

func main() {
	q := delayqueue.New(2, delayqueue.BlockOverflow)

	_ = q.Add(3*time.Second, func() { println("task 3 executed") })
	_ = q.Add(2*time.Second, func() { println("task 2 executed") })
	_ = q.Add(1*time.Second, func() { println("task 1 executed") })

	time.Sleep(4 * time.Second)
}
```

## API

### `New(cap int, policy OverflowPolicy) *delayedQueue`

Creates a delay queue:

- `cap`: maximum number of queued tasks.
- `policy`: overflow behavior when capacity is reached.

### `Add(d time.Duration, task Task) error`

Schedules `task` to run after duration `d`.

- The task runs asynchronously in its own goroutine when due.
- Returns `nil` in normal flow.

## Overflow Policies

### `BlockOverflow`

When the queue is full, `Add` blocks until space is available.

This matches `TestDelayqueueWait`:

```go
q := delayqueue.New(2, delayqueue.BlockOverflow)
```

### `DropOldest`

When the queue is full, the earliest scheduled pending task is removed, and the new task is inserted.

This matches `TestDelayqueueDrop`:

```go
q := delayqueue.New(2, delayqueue.DropOldest)
```

## Notes

- Tasks are ordered by trigger time (earliest first).
- For equal trigger times, insertion order is preserved.
- The queue is in-memory only (no persistence).
