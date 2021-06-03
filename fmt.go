package fuel

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
)

// Formatable may be used instead of fmt.Fprintf() for exactly one fmt.Formatter.
type Formatable struct {
	// Output is the actual writer.
	Output io.Writer
	// Error is the first write error.
	Error ErrorWithStack

	Wid     int
	Prec    int
	HasWid  bool
	HasPrec bool
	Flags   map[int]struct{}
}

var _ fmt.State = (*Formatable)(nil)

func (f *Formatable) Write(b []byte) (n int, err error) {
	n, err = f.Output.Write(b)
	if err != nil {
		// TODO: use own implementation
		ws := errors.WithStack(err).(ErrorWithStack)
		err = ws

		if f.Error == nil {
			f.Error = ws
		}
	}

	return
}

func (f *Formatable) Width() (int, bool) {
	return f.Wid, f.HasWid
}

func (f *Formatable) Precision() (int, bool) {
	return f.Prec, f.HasPrec
}

func (f *Formatable) Flag(c int) bool {
	_, ok := f.Flags[c]
	return ok
}
