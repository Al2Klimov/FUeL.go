package fuel

import (
	"bytes"
	"context"
	"encoding"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"
)

type Causer interface {
	Cause() error
}

type ErrorWithStack interface {
	error
	StackTracer
}

type StackTracer interface {
	StackTrace() errors.StackTrace
}

type Unwrapper interface {
	Unwrap() error
}

// AttachStackToError attaches a complete errors.StackTrace of the calling goroutine to $err if needed,
// without AttachStackToError itself and $skip additional frames at the top.
func AttachStackToError(err error, skip int) ErrorWithStack {
	if err == nil {
		return nil
	}

	if ws, ok := err.(ErrorWithStack); ok {
		return ws
	}

	return AdvancedError{
		Err: err,
		Stack: GetStack(
			1 + // AttachStackToError
				skip,
		),
	}
}

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
			stackAsRaw(stack),
		)

		if actual < frames {
			return stack[:actual]
		}

		frames *= 2
	}
}

// AdvancedError is a feature-rich error wrapper.
type AdvancedError struct {
	Err   error
	Stack errors.StackTrace
}

var _ Causer = AdvancedError{}

func (ae AdvancedError) Cause() error {
	return ae.Err
}

var _ error = AdvancedError{}

func (ae AdvancedError) Error() string {
	return ae.Err.Error()
}

var _ fmt.Formatter = AdvancedError{}

// Format appends the stack on %+v.
func (ae AdvancedError) Format(fs fmt.State, verb rune) {
	FormatNonFormatter(fs, verb, ae.Err)

	if verb == 'v' && fs.Flag('+') {
		ae.Stack.Format(fs, verb)
	}
}

var _ json.Marshaler = AdvancedError{}

func (ae AdvancedError) MarshalJSON() ([]byte, error) {
	type frame = struct {
		File     string `json:"file,omitempty"`
		Line     int    `json:"line,omitempty"`
		Function string `json:"function,omitempty"`
	}

	var err interface{}
	switch ae.Err.(type) {
	case json.Marshaler, encoding.TextMarshaler:
		err = ae.Err
	default:
		err = ae.Err.Error()
	}

	var stack []frame
	frames := runtime.CallersFrames(stackAsRaw(ae.Stack))

	for {
		fr, ok := frames.Next()
		if !ok {
			break
		}

		stack = append(stack, frame{fr.File, fr.Line, fr.Function})
	}

	return json.Marshal(struct {
		Error interface{} `json:"error"`
		Stack []frame     `json:"stack,omitempty"`
	}{err, stack})
}

var _ StackTracer = AdvancedError{}

func (ae AdvancedError) StackTrace() errors.StackTrace {
	return ae.Stack
}

var _ fmt.Stringer = AdvancedError{}

func (ae AdvancedError) String() string {
	s, _ := ae.MarshalText()
	return string(s)
}

var _ encoding.TextMarshaler = AdvancedError{}

func (ae AdvancedError) MarshalText() (text []byte, err error) {
	buf := &bytes.Buffer{}
	ae.Format(&Formatable{Output: buf, Flags: map[int]struct{}{'+': {}}}, 'v')

	return buf.Bytes(), nil
}

var _ Unwrapper = AdvancedError{}

func (ae AdvancedError) Unwrap() error {
	return ae.Err
}

// errors.StackTrace is []uintptr
var (
	_ []errors.Frame = errors.StackTrace(nil)
	_                = (*uintptr)((*errors.Frame)(nil))
)

func stackAsRaw(stack errors.StackTrace) []uintptr {
	return *(*[]uintptr)(unsafe.Pointer(&stack)) // errors.StackTrace is []uintptr
}

// ErrorGroup is a more feature-rich version of golang.org/x/sync/errgroup.Group:
//
// * enforces usage of ErrorWithStack, not just error
// * context is forwarded to tasks
// * optional concurrency limit
// * stops on context cancellation
type ErrorGroup struct {
	cancel func()
	ctx    context.Context
	err    ErrorWithStack
	once   sync.Once
	queued uintptr
	rq     RunQueue
}

// NewErrorGroup creates a new ErrorGroup. $ctx is forwarded to tasks. $concurrency < 1 means infinite.
func NewErrorGroup(ctx context.Context, concurrency int64) *ErrorGroup {
	myctx, cancel := context.WithCancel(ctx)
	eg := &ErrorGroup{cancel: cancel, ctx: myctx}

	if concurrency < 1 {
		eg.rq = NewElasticQueue(myctx)
	} else {
		eg.rq = NewLimitedQueue(myctx, concurrency)
	}

	return eg
}

func (eg *ErrorGroup) Go(weight int64, f func(context.Context) ErrorWithStack) {
	atomic.AddUintptr(&eg.queued, 1)

	eg.rq.Enqueue(weight, func(ctx context.Context) {
		atomic.AddUintptr(&eg.queued, ^uintptr(0))

		if err := f(ctx); err != nil {
			eg.once.Do(func() {
				eg.err = err
				eg.cancel()
			})
		}
	})
}

func (eg *ErrorGroup) Wait() ErrorWithStack {
	eg.rq.Wait()

	if eg.err == nil && atomic.LoadUintptr(&eg.queued) > 0 {
		return AttachStackToError(eg.ctx.Err(), 0)
	}

	return eg.err
}
