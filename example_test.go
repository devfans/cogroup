package cogroup_test

import (
	"context"
	"time"

	"github.com/devfans/cogroup"
)

func ExampleStart() {
	f := func(context.Context) error {
		<-time.After(time.Second)
		return nil
	}

	g := cogroup.Start(context.Background(), 2, 10, false)
	for i := 0; i < 10; i++ {
		g.Add(f)
	}
	g.Wait()
}

func ExampleStart_startWithCancelContext() {
	f := func(ctx context.Context) error {
		<-time.After(time.Second)
		workerID := cogroup.GetWorkerID(ctx)
		println(workerID, " did one task")
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	g := cogroup.Start(ctx, 2, 10, false)
	go func() {
		<-time.After(1 * time.Second)
		cancel()
	}()

	for i := 0; i < 100; i++ {
		g.Add(f)
	}
	println("Tasks left:", g.Wait())
}
