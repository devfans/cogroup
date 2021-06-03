package cogroup

import (
	"context"
	"testing"
	"time"
)

func Test_CoGroup(t *testing.T) {
	f := func(ctx context.Context) error {
		<-time.After(time.Second)
		t.Log(GetWorkerID(ctx), "xxxxxxxxxxxxxxxxxx")
		return nil
	}

	g := Start(context.Background(), 2, 10, false)
	for i := 0; i < 1; i++ {
		g.Add(f)
	}
	a := g.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	g = Start(ctx, 2, 10, false)
	go func() {
		<-time.After(1 * time.Second)
		cancel()
	}()
	for i := 0; i < 20; i++ {
		g.Add(f)
	}
	b := g.Wait()
	if a != 0 || b == 0 {
		t.Error("Unexpect queue length", a, b)
	}
}
