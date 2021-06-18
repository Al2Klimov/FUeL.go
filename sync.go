package fuel

import (
	"context"
	"golang.org/x/sync/semaphore"
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

// LimitedQueue runs enqueued tasks with limited concurrency in FIFO order until context cancellation.
type LimitedQueue struct {
	eq    ElasticQueue
	items []queueItem
	mtx   sync.Mutex
	sema  *semaphore.Weighted
}

// NewLimitedQueue creates a new LimitedQueue which runs $concurrency tasks at a time.
// $ctx is forwarded to Enqueue()d tasks.
func NewLimitedQueue(ctx context.Context, concurrency int64) *LimitedQueue {
	return &LimitedQueue{
		eq:   ElasticQueue{ctx: ctx},
		sema: semaphore.NewWeighted(concurrency),
	}
}

var _ RunQueue = (*LimitedQueue)(nil)

func (lq *LimitedQueue) Enqueue(weight int64, f func(context.Context)) {
	select {
	case <-lq.eq.ctx.Done():
		return
	default:
	}

	lq.mtx.Lock()

	if len(lq.items) < 1 && lq.sema.TryAcquire(weight) {
		lq.mtx.Unlock()
		lq.forward(weight, f)
	} else {
		lq.items = append(lq.items, queueItem{weight, f})
		lq.mtx.Unlock()
	}
}

func (lq *LimitedQueue) Wait() {
	lq.eq.Wait()
}

func (lq *LimitedQueue) forward(weight int64, f func(context.Context)) {
	lq.eq.Enqueue(weight, func(ctx context.Context) {
		defer lq.nextOnes()
		defer lq.sema.Release(weight)

		f(ctx)
	})
}

func (lq *LimitedQueue) nextOnes() {
	select {
	case <-lq.eq.ctx.Done():
		return
	default:
	}

	lq.mtx.Lock()

	for len(lq.items) > 0 {
		if next := lq.items[0]; lq.sema.TryAcquire(next.weight) {
			lq.forward(next.weight, next.f)
			lq.items = lq.items[1:]
		} else {
			break
		}
	}

	lq.mtx.Unlock()
}

type queueItem struct {
	weight int64
	f      func(context.Context)
}
