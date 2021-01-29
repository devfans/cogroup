package cogroup

import (
	"context"
	"errors"
	"runtime"
	"sync"
  "fmt"
)

// Coroutine group controller
// - panic recover
// - support upstream cancel func
// - max procs control
// NOTE：Add() will block if the job buffer is full

var GROUP_CLOSED_ERROR = errors.New("Group access was already closed")

type CoGroup struct {
	context.Context
	wg   sync.WaitGroup
	ch   chan func(context.Context) error // Task chan
	sink bool                             // Use group context or not
	open bool                             // Open signal
	jobs int
	done chan bool // Close chan for draining
	sync.Mutex
}

// Intitialize a new run group
// - n: max goroutines
// - m: jobs buffer size
func RunGroup(ctx context.Context, n uint, m uint, sink bool) *CoGroup {
	g := &CoGroup{
		Context: ctx,
		ch:      make(chan func(context.Context) error, m),
		done:    make(chan bool, m),
		sink:    sink,
		open:    true,
	}
	g.start(int(n))
	return g
}

// Add may block if jobs buffer is full
func (g *CoGroup) Add(f func(context.Context) error) error {
	g.Lock()
	defer g.Unlock()
	open := g.open
	if open {
		g.ch <- f
		g.jobs++
		return nil
	}
	return GROUP_CLOSED_ERROR
}

func (g *CoGroup) start(n int) {
	for i := 0; i < n; i++ {
		g.wg.Add(1)
		go g.process()
	}
}

func (g *CoGroup) process() {
	defer g.wg.Done()
	for {
		select {
		case f, ok := <-g.ch:
			if !ok {
				return
			}
			g.run(f)
		case <-g.Done():
			return
		}
	}
}

func (g *CoGroup) run(f func(context.Context) error) {
	defer func() {
		if r := recover(); r != nil {
			err := make([]byte, 200)
			err = err[:runtime.Stack(err, false)]
			fmt.Printf("CoGroup panic captured %v %s\n", r, err)
		}
	}()

	if g.sink {
		f(g.Context)
	} else {
		f(context.Background())
	}
	go func() {
		g.done <- true
	}()
	return
}

func (g *CoGroup) Wait() {
	g.Lock()
	g.open = false
	n := g.jobs
	g.Unlock()
	go func() {
		defer close(g.ch)
		for i := 0; i < n; i++ {
			select {
			case <-g.done:
			case <-g.Done():
				return
			}
		}
	}()

	g.wg.Wait()
}
