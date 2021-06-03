package fuel

import (
	"fmt"
	"github.com/pkg/errors"
	"strings"
	"testing"
)

func TestErrorWithStack(t *testing.T) {
	if _, ok := errors.New("").(ErrorWithStack); !ok {
		t.Error("ErrorWithStack doesn't cover github.com/pkg/errors errors")
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

func recurse(steps uint8, finally func()) {
	if steps > 0 {
		recurse(steps-1, finally)
	} else {
		finally()
	}
}
