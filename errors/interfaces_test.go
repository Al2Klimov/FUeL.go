package errors

import (
	"github.com/pkg/errors"
	"testing"
)

func TestWithStack(t *testing.T) {
	if _, ok := errors.New("").(WithStack); !ok {
		t.Error("WithStack doesn't cover github.com/pkg/errors errors")
	}
}
