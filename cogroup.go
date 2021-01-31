// Package cogroup provides a elegant goroutine group with context controls. It's designed to meet the following requirements.
//
// - Tasks can be executed with order
//
// - Group `wait` command will close the write acces to the task queue
//
// - Upstream context can cancel the task queue
//
// - When the context is canceled, the tasks in queue will be no longer consumed
//
// - Panic recover for a single task execution
//
// - Only spawn specified number of goroutines to consume the task
//
// - `Wait` will return block until tasks are finished or canceled, and return with the queue length
//
package cogroup

import (
	"context"
	"fmt"
	"runtime"
	"sync"
)

// Coroutine group struct holds the group state: the task queue, context and signals.
type CoGroup struct {
	ctx context.Context                   // Group context
	wg   sync.WaitGroup                   // Group goroutine wait group
	ch   chan func(context.Context) error // Task chan
	sink bool                             // Use group context or not
}

// Start will initialize a cogroup and start the group goroutines.
// Parameter `n` specifies the number the goroutine to start as workers to consume the task queue.
// Parameter `m` specifies the size of the task queue buffer, if the buffer is full, the `Insert` method will block till there's more room or a cancel signal was received.
// Parameter `sink` specifies whether to pass the group context to the task.
func Start(ctx context.Context, n uint, m uint, sink bool) *CoGroup {
	g := &CoGroup{
		ctx: ctx,
		ch:      make(chan func(context.Context) error, m),
		sink:    sink,
	}
	g.start(int(n))
	return g
}

// Add a task into the task queue without blocking.
func (g *CoGroup) Add(f func(context.Context) error) {
	select {
	case g.ch <- f:
	default:
		go g.Insert(f)
	}
}

// Insert a task into the task queue with blocking if the task queue buffer is full.
// If the group context was canceled already, it will abort with a false return.
func (g *CoGroup) Insert(f func(context.Context) error) (success bool) {
	defer func() {
		if r := recover(); r != nil {
			success = false
		}
	}()
	select {
	case g.ch <- f:
		success = true
	case <-g.ctx.Done():
	}
	return
}

// Start the coroutine group
func (g *CoGroup) start(n int) {
	for i := 0; i < n; i++ {
		g.wg.Add(1)
		go g.process()
	}
}

// Start a single coroutine
func (g *CoGroup) process() {
	defer g.wg.Done()
	for {
		select {
		case <-g.ctx.Done():
			return
		default:
			select {
			case f, ok := <-g.ch:
				if !ok {
					return
				}
				g.run(f)
			case <-g.ctx.Done():
				return
			}
		}
	}
}

// Execute a single task
func (g *CoGroup) run(f func(context.Context) error) {
	defer func() {
		if r := recover(); r != nil {
			err := make([]byte, 200)
			err = err[:runtime.Stack(err, false)]
			fmt.Printf("CoGroup panic captured %v %s\n", r, err)
		}
	}()

	if g.sink {
		f(g.ctx)
	} else {
		f(context.Background())
	}
	return
}

// Size return the current length the task queue
func (g *CoGroup) Size() int {
	return len(g.ch)
}

// Wait till the tasks in queue are all finished, or the group was canceled by the context.
func (g *CoGroup) Wait() int {
	close(g.ch)
	g.wg.Wait()
	return len(g.ch)
}

// Reset the cogroup, it will call the group `Wait` first before do a internal reset.
func (g *CoGroup) Reset() {
	g.Wait()
	g.ch = make(chan func(context.Context) error, cap(g.ch))
}
