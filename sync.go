package fuel

import (
	"context"
	"sync"
)

type RunQueue interface {
	// Enqueue enqueues the task $f for being run ASAP. $f is counted as $weight item(s).
	Enqueue(weight int64, f func(context.Context))
	// Wait waits for all Enqueue()d tasks to finish.
	Wait()
}

// ElasticQueue runs enqueued tasks immediately until context cancellation.
type ElasticQueue struct {
	ctx context.Context
	wg  sync.WaitGroup
}

// NewElasticQueue creates a new ElasticQueue. $ctx is forwarded to Enqueue()d tasks.
func NewElasticQueue(ctx context.Context) *ElasticQueue {
	return &ElasticQueue{ctx: ctx}
}

var _ RunQueue = (*ElasticQueue)(nil)

func (eq *ElasticQueue) Enqueue(_ int64, f func(context.Context)) {
	select {
	case <-eq.ctx.Done():
		return
	default:
	}

	eq.wg.Add(1)

	go func() {
		defer eq.wg.Done()

		f(eq.ctx)
	}()
}

func (eq *ElasticQueue) Wait() {
	eq.wg.Wait()
}
