package fuel

import (
	"github.com/pkg/errors"
	"runtime"
	"unsafe"
)

type ErrorWithStack interface {
	error
	StackTracer
}

type StackTracer interface {
	StackTrace() errors.StackTrace
}

// errors.StackTrace is []uintptr
var (
	_ []errors.Frame = errors.StackTrace(nil)
	_                = (*uintptr)((*errors.Frame)(nil))
)

// GetStack returns a complete errors.StackTrace of the calling goroutine
// without GetStack itself and $skip additional frames at the top.
func GetStack(skip int) errors.StackTrace {
	frames := 32
	for {
		stack := make(errors.StackTrace, frames)

		actual := runtime.Callers(
			1+ // runtime.Callers
				1+ // GetStack
				skip,
			*(*[]uintptr)(unsafe.Pointer(&stack)), // errors.StackTrace is []uintptr
		)

		if actual < frames {
			return stack[:actual]
		}

		frames *= 2
	}
}
