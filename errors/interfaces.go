package errors

import "github.com/pkg/errors"

type StackTracer interface {
	StackTrace() errors.StackTrace
}

type WithStack interface {
	error
	StackTracer
}
