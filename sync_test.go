package fuel

import (
	"context"
	"testing"
	"time"
)

func TestElasticQueue(t *testing.T) {
	const concurrency = 16
	ctx, cancel := context.WithCancel(context.Background())
	queue := NewElasticQueue(ctx)

	assertTakesTime(t, time.Second, time.Second/10, func() {
		for i := 0; i < concurrency; i++ {
			queue.Enqueue(1, dumbSleeper(time.Second))
		}

		queue.Wait()
	})

	assertTakesTime(t, time.Second/2, time.Second/10, func() {
		for i := 0; i < concurrency; i++ {
			queue.Enqueue(1, smartSleeper(time.Second))
		}

		time.Sleep(time.Second / 2)
		cancel()
		queue.Wait()
	})

	assertTakesTime(t, 0, time.Second/10, func() {
		for i := 0; i < concurrency; i++ {
			queue.Enqueue(1, dumbSleeper(time.Second))
		}

		queue.Wait()
	})
}

func TestLimitedQueue(t *testing.T) {
	const items = 16
	ctx, cancel := context.WithCancel(context.Background())
	queue := NewLimitedQueue(ctx, 4)

	assertTakesTime(t, 8*time.Second, time.Second/10, func() {
		for i := 0; i < items; i++ {
			queue.Enqueue(2, dumbSleeper(time.Second))
		}

		queue.Wait()
	})

	assertTakesTime(t, 4*time.Second, time.Second/10, func() {
		for i := 0; i < items; i++ {
			queue.Enqueue(2, smartSleeper(time.Second))
		}

		time.Sleep(4 * time.Second)
		cancel()
		queue.Wait()
	})

	assertTakesTime(t, 0, time.Second/10, func() {
		for i := 0; i < items; i++ {
			queue.Enqueue(2, dumbSleeper(time.Second))
		}

		queue.Wait()
	})
}

func assertTakesTime(t *testing.T, dur, latency time.Duration, f func()) {
	t.Helper()

	start := time.Now()
	f()

	if actual := time.Since(start); actual < dur || actual > dur+latency {
		t.Errorf("function took %s, expected [%s, %s]", actual, dur, dur+latency)
	}
}

func dumbSleeper(dur time.Duration) func(context.Context) {
	return func(context.Context) {
		time.Sleep(dur)
	}
}

func smartSleeper(dur time.Duration) func(context.Context) {
	return func(ctx context.Context) {
		timer := time.NewTimer(dur)
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
}
