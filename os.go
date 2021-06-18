package fuel

import (
	"context"
	"os"
	"os/signal"
)

// SignalsToContext derives $child from $ctx and cancels it on one of $signals.
// The exact signal (or nil on $ctx cancellation) is forwarded to $reason.
// $signals will be handled by SignalsToContext until $ctx cancellation to prevent firing of default handlers.
// Cancel $ctx not to leak goroutines!
func SignalsToContext(ctx context.Context, signals ...os.Signal) (child context.Context, reason <-chan os.Signal) {
	myctx, cancel := context.WithCancel(ctx)
	in := make(chan os.Signal, 1)
	out := make(chan os.Signal, 1)

	signal.Notify(in, signals...)

	go func() {
		select {
		case <-ctx.Done():
			signal.Stop(in)
			out <- nil
		case s := <-in:
			out <- s
			cancel()

			<-ctx.Done()
			signal.Stop(in)
		}
	}()

	return myctx, out
}
