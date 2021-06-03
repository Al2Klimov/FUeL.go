package fuel

import "github.com/pkg/errors"

type ErrorWithStack interface {
	error
	StackTracer
}

type StackTracer interface {
	StackTrace() errors.StackTrace
}
