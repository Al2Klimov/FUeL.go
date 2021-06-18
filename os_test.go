package fuel

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestSignalsToContext(t *testing.T) {
	{
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		child, reason := SignalsToContext(ctx, syscall.SIGUSR1)
		var ctxErr error
		var actual os.Signal

		syscall.Kill(syscall.Getpid(), syscall.SIGUSR1)

		select {
		case <-child.Done():
		case <-time.After(time.Second / 2):
		}

		ctxErr = child.Err()

		select {
		case actual = <-reason:
		default:
		}

		if ctxErr != context.Canceled || actual != syscall.SIGUSR1 {
			t.Errorf("SignalsToContext: got %#v,%#v, expected context.Canceled,syscall.SIGUSR1", ctxErr, actual)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	child, reason := SignalsToContext(ctx, syscall.SIGUSR1)
	var ctxErr error
	var actual os.Signal

	cancel()

	select {
	case <-child.Done():
	case <-time.After(time.Second / 2):
	}

	ctxErr = child.Err()

	select {
	case actual = <-reason:
	default:
	}

	if ctxErr != context.Canceled || actual != nil {
		t.Errorf("SignalsToContext: got %#v,%#v, expected context.Canceled,nil", ctxErr, actual)
	}
}
