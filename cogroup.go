// Package cogroup provides a elegant goroutine group with context controls. It's designed to meet the following requirements.
//
// - Tasks can be executed without order
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
// - `Wait` will block until tasks are finished or canceled, and return with the queue length
//
package cogroup

import (
	"context"
	"fmt"
	"os"
	"runtime/debug"
	"sync"
)

// Worker CoGroup Worker factory
//
// `ctx` is provided from the cogroup
//
// `i` indicates the worker id
//
// `f` is job to consume
type Worker func(ctx context.Context, i int, f func(context.Context) error)

// CoGroup Coroutine group struct holds the group state: the task queue, context and signals.
type CoGroup struct {
	worker Worker
	ctx    context.Context                  // Group context
	wg     sync.WaitGroup                   // Group goroutine wait group
	ch     chan func(context.Context) error // Task chan
	sink   bool                             // Use group context or not
	n      int                              // Number of workers to spawn
}

// Worker meta context key
type workerKey struct{}

// New will create a cogroup instance without starting the group
func New(ctx context.Context, n uint, m uint, sink bool) *CoGroup {
	if n < 1 {
		panic("At least one goroutine should spawned in cogroup!")
	}
	return &CoGroup{
		ctx:  ctx,
		ch:   make(chan func(context.Context) error, m),
		n:    int(n),
		sink: sink,
	}
}

// Start will initialize a cogroup and start the group goroutines.
//
// Parameter `n` specifies the number the goroutine to start as workers to consume the task queue.
//
// Parameter `m` specifies the size of the task queue buffer, if the buffer is full, the `Insert` method will block till there's more room or a cancel signal was received.
//
// Parameter `sink` specifies whether to pass the group context to the task.
func Start(ctx context.Context, n uint, m uint, sink bool) *CoGroup {
	g := New(ctx, n, m, sink)
	g.worker = g.run
	g.start(g.n)
	return g
}

// StartWithWorker will register customized worker and start the group goroutines
//
// If worker is `nil`, the default plain worker will be used.
func (g *CoGroup) StartWithWorker(worker Worker) {
	if worker == nil {
		g.worker = g.run
	} else {
		g.worker = worker
	}
	g.start(g.n)
}

// TryInsert without blocking will return false when the task queue is full or closed, or the context was canceled already.
func (g *CoGroup) TryInsert(f func(context.Context) error) (success bool) {
	defer func() {
		recover()
	}()
	select {
	case g.ch <- f:
		success = true
	case <-g.ctx.Done():
	default:
	}
	return
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
		recover()
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
		go g.process(i)
	}
}

// Start a single coroutine
func (g *CoGroup) process(i int) {
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
				g.worker(g.ctx, i, f)
			case <-g.ctx.Done():
				return
			}
		}
	}
}

// Execute a single task
func (g *CoGroup) run(_ context.Context, i int, f func(context.Context) error) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Fprintf(os.Stderr, "CoGroup panic captured: %v - %s", err, debug.Stack())
		}
	}()

	if g.sink {
		f(context.WithValue(g.ctx, workerKey{}, i))
	} else {
		f(context.WithValue(context.Background(), workerKey{}, i))
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

// GetWorkers Get the number of total group workers
func (g *CoGroup) GetWorkers() int {
	return g.n
}

// GetWorkerID Get worker id from the context
func GetWorkerID(ctx context.Context) int {
	n, _ := ctx.Value(workerKey{}).(int)
	return n
}
