package cogroup

import (
	"context"
	"testing"
	"time"
)

func Test_CoGroup(t *testing.T) {
	f := func(context.Context) error {
		<-time.After(time.Second)
		t.Log("xxxxxxxxxxxxxxxxxx")
		return nil
	}

	g := RunGroup(context.Background(), 2, 10, false)
	for i := 0; i < 10; i++ {
		g.Add(f)
	}
	g.Wait()

	ctx, cancel := context.WithCancel(context.Background())
	g = RunGroup(ctx, 2, 100, false)
	go func() {
		<-time.After(10 * time.Second)
		cancel()
	}()
	for i := 0; i < 100; i++ {
		g.Add(f)
	}
	g.Wait()
}
