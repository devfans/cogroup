# cogroup
Golang coroutine group

[![Build Status](https://travis-ci.org/devfans/cogroup.svg?branch=master)](https://travis-ci.org/devfans/cogroup)
[![Go Report Card](https://goreportcard.com/badge/github.com/devfans/cogroup)](https://goreportcard.com/report/github.com/devfans/cogroup)
[![GoDoc](https://godoc.org/github.com/devfans/cogroup?status.svg)](https://godoc.org/github.com/devfans/cogroup) [![Join the chat at https://gitter.im/devfans/envconf](https://badges.gitter.im/devfans/cogroup.svg)](https://gitter.im/devfans/cogroup?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

Package cogroup provides a elegant goroutine group with context controls. It's designed to meet the following requirements.

- Tasks can be executed with order
- Group `wait` command will close the write acces to the task queue
- Upstream context can cancel the task queue
- When the context is canceled, the tasks in queue will be no longer consumed
- Only spawn specified number of goroutines to consume the task
- Panic recover for a single task execution
- `Wait` will return block until tasks are finished or canceled, and return with the queue length

### Usage

Start a group and wait till all the tasks are finished.

```
import  (
  "context"
  "time"

  "github.com/devfans/cogroup"
)

func main() {
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

```

Start a group and cancel it later.

```
import  (
  "context"
  "time"

  "github.com/devfans/cogroup"
)

func main() {
  f := func(ctx context.Context) error {
    <-time.After(time.Second)
    workerId := cogroup.GetWorkerId(ctx)
    println(workerId, " did one task")
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

```


### Misc

Blog：https://blog.devfans.io/create-a-golang-coroutine-group/
