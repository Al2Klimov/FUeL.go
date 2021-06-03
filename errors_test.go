package fuel

import (
	"github.com/pkg/errors"
	"testing"
)

func TestErrorWithStack(t *testing.T) {
	if _, ok := errors.New("").(ErrorWithStack); !ok {
		t.Error("ErrorWithStack doesn't cover github.com/pkg/errors errors")
	}
}
