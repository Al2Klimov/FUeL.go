package fuel

import (
	"bytes"
	"context"
	"encoding"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestErrorWithStack(t *testing.T) {
	if _, ok := errors.New("").(ErrorWithStack); !ok {
		t.Error("ErrorWithStack doesn't cover github.com/pkg/errors errors")
	}
}

func TestAttachStackToError(t *testing.T) {
	if actual := AttachStackToError(nil, 0); actual != nil {
		t.Errorf("AttachStackToError(nil, 0): got %#v, expected nil", actual)
	}

	thirdParty1 := errors.New("")
	thirdParty2 := errors.New("")

	if thirdParty1 == thirdParty2 {
		t.Error("third-party errors with stacks are not distinct")
	} else {
		if actual := AttachStackToError(thirdParty1, 0); actual != thirdParty1 {
			t.Errorf("AttachStackToError(%#v, 0): got %#v, expected %#v", thirdParty1, actual, thirdParty1)
		}
	}

	err := io.EOF
	actual := AttachStackToError(err, 0)

	if ae, ok := actual.(AdvancedError); ok {
		if ae.Err != err {
			t.Errorf("AttachStackToError(%#v, 0).Err: got %#v, expected %#v", err, ae.Err, err)
		}

		if stack := GetStack(0); len(ae.Stack) != len(stack) {
			t.Errorf("AttachStackToError(%#v, 0).Stack: got %d frames, expected %d", err, len(ae.Stack), len(stack))
		}
	} else {
		t.Errorf("AttachStackToError(%#v, 0): got %#v, expected an AdvancedError", err, actual)
	}
}

func TestGetStack(t *testing.T) {
	if stack := GetStack(0); len(stack) < 1 {
		t.Error("GetStack(0): stack empty")
	} else {
		for _, ptr := range stack {
			if ptr == 0 {
				t.Error("GetStack(0): has a 0x0 frame")
			}
		}

		actual := fmt.Sprintf("%n", stack[0])
		if !strings.Contains(actual, "TestGetStack") {
			t.Errorf("GetStack(0): got %s on top of the stack, expected TestGetStack", actual)
		}
	}

	if stack := GetStack(1); len(stack) > 0 {
		actual := fmt.Sprintf("%n", stack[0])
		if strings.Contains(actual, "TestGetStack") {
			t.Error("GetStack(1): got TestGetStack on top of the stack")
		}
	}

	const expected = 64
	var actual errors.StackTrace

	recurse(expected, func() { actual = GetStack(0) })

	if len(actual) < expected {
		t.Errorf("GetStack(0): got %d frames, expected >=%d", len(actual), expected)
	}
}

func TestAdvancedError_Format(t *testing.T) {
	const dummy = "42"

	assertAdvancedError_Format(
		t, testFormatterError{testFormatter{dummy}, pseudoError{}}, &Formatable{Wid: 2, HasWid: true}, 'v',
		func(actual []byte) string {
			const expected = "self=42, state=2, verb=v"
			if string(actual) == expected {
				return ""
			} else {
				return fmt.Sprintf(" expected %#v", expected)
			}
		},
	)

	assertAdvancedError_Format(
		t, testFormatterError{testFormatter{dummy}, pseudoError{}},
		&Formatable{Wid: 2, HasWid: true, Flags: map[int]struct{}{'+': {}}}, 's',
		func(actual []byte) string {
			const expected = "self=42, state=+2, verb=s"
			if string(actual) == expected {
				return ""
			} else {
				return fmt.Sprintf(" expected %#v", expected)
			}
		},
	)

	assertAdvancedError_Format(
		t, io.EOF, &Formatable{Flags: map[int]struct{}{'+': {}}}, 'v',
		func(actual []byte) string {
			if lines := bytes.Count(actual, []byte{'\n'}); lines < 64 {
				return ", too few lines"
			} else {
				return ""
			}
		},
	)
}

func assertAdvancedError_Format(t *testing.T, err error, fs *Formatable, verb rune, validator func([]byte) string) {
	t.Helper()

	buf := &bytes.Buffer{}
	fs.Output = buf

	var stack errors.StackTrace
	recurse(64, func() {
		stack = GetStack(0)
	})

	AdvancedError{err, stack}.Format(fs, verb)

	if reason := validator(buf.Bytes()); reason != "" {
		t.Errorf("AdvancedError{%#v, %#v}.Format(%#v, '%c'): got %#v%s", err, stack, fs, verb, buf.String(), reason)
	}
}

func TestAdvancedError_MarshalJSON(t *testing.T) {
	err1 := io.EOF
	stack := GetStack(0)

	if jsn, err := (AdvancedError{err1, stack}.MarshalJSON()); err == nil {
		var tree interface{}
		if err := json.Unmarshal(jsn, &tree); err == nil {
			if root, ok := tree.(map[string]interface{}); ok {
				if errorBranch, ok := root["error"]; ok {
					if errorString, ok := errorBranch.(string); ok {
						if errorString != err1.Error() {
							t.Errorf(
								"AdvancedError{%#v, %#v}.MarshalJSON(): got .error %#v, expected %#v",
								err1, stack, errorString, err1.Error(),
							)
						}
					} else {
						t.Errorf(
							"AdvancedError{%#v, %#v}.MarshalJSON(): got %#v, .error is not a string",
							err1, stack, string(jsn),
						)
					}
				} else {
					t.Errorf("AdvancedError{%#v, %#v}.MarshalJSON(): got %#v, .error missing", err1, stack, string(jsn))
				}

				if stackBranch, ok := root["stack"]; ok {
					if stackArray, ok := stackBranch.([]interface{}); ok {
						if len(stackArray) < 1 {
							t.Errorf(
								"AdvancedError{%#v, %#v}.MarshalJSON(): got %#v, .stack is empty",
								err1, stack, string(jsn),
							)
						} else if frameObject, ok := stackArray[0].(map[string]interface{}); ok {
							if fileBranch, ok := frameObject["file"]; ok {
								if _, ok := fileBranch.(string); !ok {
									t.Errorf(
										"AdvancedError{%#v, %#v}.MarshalJSON(): got %#v,"+
											" .stack[0].file is not a string",
										err1, stack, string(jsn),
									)
								}
							} else {
								t.Errorf(
									"AdvancedError{%#v, %#v}.MarshalJSON(): got %#v, .stack[0].file missing",
									err1, stack, string(jsn),
								)
							}

							if lineBranch, ok := frameObject["line"]; ok {
								if _, ok := lineBranch.(float64); !ok {
									t.Errorf(
										"AdvancedError{%#v, %#v}.MarshalJSON(): got %#v,"+
											" .stack[0].line is not a number",
										err1, stack, string(jsn),
									)
								}
							} else {
								t.Errorf(
									"AdvancedError{%#v, %#v}.MarshalJSON(): got %#v, .stack[0].line missing",
									err1, stack, string(jsn),
								)
							}

							if functionBranch, ok := frameObject["function"]; ok {
								if _, ok := functionBranch.(string); !ok {
									t.Errorf(
										"AdvancedError{%#v, %#v}.MarshalJSON(): got %#v,"+
											" .stack[0].function is not a string",
										err1, stack, string(jsn),
									)
								}
							} else {
								t.Errorf(
									"AdvancedError{%#v, %#v}.MarshalJSON(): got %#v, .stack[0].function missing",
									err1, stack, string(jsn),
								)
							}
						} else {
							t.Errorf(
								"AdvancedError{%#v, %#v}.MarshalJSON(): got %#v, .stack[0] is not an object",
								err1, stack, string(jsn),
							)
						}
					} else {
						t.Errorf(
							"AdvancedError{%#v, %#v}.MarshalJSON(): got %#v, .stack is not an array",
							err1, stack, string(jsn),
						)
					}
				} else {
					t.Errorf("AdvancedError{%#v, %#v}.MarshalJSON(): got %#v, .stack missing", err1, stack, string(jsn))
				}
			} else {
				t.Errorf(
					"AdvancedError{%#v, %#v}.MarshalJSON(): got %#v, root is not an object",
					err1, stack, string(jsn),
				)
			}
		} else {
			t.Errorf(
				"AdvancedError{%#v, %#v}.MarshalJSON(): got bad JSON %#v: %s",
				err1, stack, string(jsn), err.Error(),
			)
		}
	} else {
		t.Errorf("AdvancedError{%#v, %#v}.MarshalJSON(): got %#v, expected nil", err1, stack, err)
	}

	err2 := testJsonError{jsn: []byte("1e42")}
	if jsn, err := (AdvancedError{err2, nil}.MarshalJSON()); err == nil {
		const expected = `{"error":1e42}`
		if bytes.Compare(jsn, []byte(expected)) != 0 {
			t.Errorf("AdvancedError{%#v, nil}.MarshalJSON(): got %#v, expected %#v", err2, string(jsn), expected)
		}
	} else {
		t.Errorf("AdvancedError{%#v, nil}.MarshalJSON(): got %#v, expected nil", err2, err)
	}

	err3 := testTextError{text: []byte("42")}
	if jsn, err := (AdvancedError{err3, nil}.MarshalJSON()); err == nil {
		const expected = `{"error":"42"}`
		if bytes.Compare(jsn, []byte(expected)) != 0 {
			t.Errorf("AdvancedError{%#v, nil}.MarshalJSON(): got %#v, expected %#v", err3, string(jsn), expected)
		}
	} else {
		t.Errorf("AdvancedError{%#v, nil}.MarshalJSON(): got %#v, expected nil", err3, err)
	}
}

func TestErrorGroup(t *testing.T) {
	const items = 16

	{
		ctx, cancel := context.WithCancel(context.Background())
		eg := NewErrorGroup(ctx, 0)

		assertTakesTime(t, time.Second, time.Second/10, func() {
			for i := 0; i < items; i++ {
				eg.Go(1, errorGroupify(dumbSleeper(time.Second), nil))
			}

			eg.Wait()
		})

		assertTakesTime(t, time.Second/2, time.Second/10, func() {
			for i := 0; i < items; i++ {
				eg.Go(1, errorGroupify(smartSleeper(time.Second), nil))
			}

			time.Sleep(time.Second / 2)
			cancel()
			eg.Wait()
		})

		assertTakesTime(t, 0, time.Second/10, func() {
			for i := 0; i < items; i++ {
				eg.Go(1, errorGroupify(dumbSleeper(time.Second), nil))
			}

			eg.Wait()
		})
	}

	{
		ctx, cancel := context.WithCancel(context.Background())
		eg := NewErrorGroup(ctx, 4)

		assertTakesTime(t, 8*time.Second, time.Second/10, func() {
			for i := 0; i < items; i++ {
				eg.Go(2, errorGroupify(dumbSleeper(time.Second), nil))
			}

			eg.Wait()
		})

		assertTakesTime(t, 4*time.Second, time.Second/10, func() {
			for i := 0; i < items; i++ {
				eg.Go(2, errorGroupify(smartSleeper(time.Second), nil))
			}

			time.Sleep(4 * time.Second)
			cancel()
			eg.Wait()
		})

		assertTakesTime(t, 0, time.Second/10, func() {
			for i := 0; i < items; i++ {
				eg.Go(2, errorGroupify(dumbSleeper(time.Second), nil))
			}

			eg.Wait()
		})
	}

	{
		eg := NewErrorGroup(context.Background(), 0)

		eg.Go(1, func(context.Context) ErrorWithStack {
			time.Sleep(time.Second)
			return nil
		})

		eg.Go(1, func(context.Context) ErrorWithStack {
			time.Sleep(2 * time.Second)
			return AttachStackToError(io.EOF, 0)
		})

		eg.Go(1, func(context.Context) ErrorWithStack {
			time.Sleep(3 * time.Second)
			return AttachStackToError(io.ErrClosedPipe, 0)
		})

		actual := eg.Wait()
		if ae, ok := actual.(AdvancedError); !ok || ae.Err != io.EOF {
			t.Errorf("ErrorGroup#Wait(): got %#v, expected AdvancedError{Err: io.EOF}", actual)
		}
	}

	assertTakesTime(t, time.Second, time.Second/10, func() {
		eg := NewErrorGroup(context.Background(), 0)

		eg.Go(1, errorGroupify(dumbSleeper(time.Second), io.EOF))
		eg.Go(1, errorGroupify(smartSleeper(2*time.Second), nil))
		eg.Wait()
	})

	ctx, cancel := context.WithCancel(context.Background())
	eg := NewErrorGroup(ctx, 0)

	eg.Go(1, errorGroupNoop)
	cancel()
	eg.Go(1, errorGroupNoop)

	actual := eg.Wait()
	if ae, ok := actual.(AdvancedError); !ok || ae.Err != context.Canceled {
		t.Errorf("ErrorGroup#Wait(): got %#v, expected AdvancedError{Err: context.Canceled}", actual)
	}
}

func recurse(steps uint8, finally func()) {
	if steps > 0 {
		recurse(steps-1, finally)
	} else {
		finally()
	}
}

func errorGroupify(f func(context.Context), err error) func(context.Context) ErrorWithStack {
	return func(ctx context.Context) ErrorWithStack {
		f(ctx)
		return AttachStackToError(err, 0)
	}
}

func errorGroupNoop(context.Context) ErrorWithStack {
	return nil
}

type pseudoError struct {
}

var _ error = pseudoError{}

func (pseudoError) Error() string {
	panic("don't call me")
}

type testFormatterError struct {
	testFormatter
	pseudoError
}

var (
	_ fmt.Formatter = testFormatterError{}
	_ error         = testFormatterError{}
)

type testJsonError struct {
	pseudoError

	jsn []byte
}

var _ error = testJsonError{}

var _ json.Marshaler = testJsonError{}

func (tje testJsonError) MarshalJSON() ([]byte, error) {
	return tje.jsn, nil
}

type testTextError struct {
	pseudoError

	text []byte
}

var _ error = testTextError{}

var _ encoding.TextMarshaler = testTextError{}

func (tte testTextError) MarshalText() ([]byte, error) {
	return tte.text, nil
}
