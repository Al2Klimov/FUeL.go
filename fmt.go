package fuel

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"strconv"
)

// FmtStateToString converts $fs to its fmt.Printf() representation (see unit tests).
func FmtStateToString(fs fmt.State) string {
	format := &bytes.Buffer{}

	for _, flag := range []byte{'+', '-', '#', ' ', '0'} {
		if fs.Flag(int(flag)) {
			format.WriteByte(flag)
		}
	}

	if width, ok := fs.Width(); ok {
		format.WriteString(strconv.FormatInt(int64(width), 10))
	}

	if precision, ok := fs.Precision(); ok {
		format.WriteByte('.')
		format.WriteString(strconv.FormatInt(int64(precision), 10))
	}

	return format.String()
}

// FormatNonFormatter forwards $fs and $verb to $nonFormatter.Format() if $nonFormatter is a fmt.Formatter.
// Otherwise it formats $nonFormatter via fmt.Fprintf() as specified by $fs and $verb.
func FormatNonFormatter(fs fmt.State, verb rune, nonFormatter interface{}) {
	if formatter, ok := nonFormatter.(fmt.Formatter); ok {
		formatter.Format(fs, verb)
	} else {
		format := &bytes.Buffer{}

		format.WriteByte('%')
		format.WriteString(FmtStateToString(fs))
		format.WriteRune(verb)

		fmt.Fprintf(fs, format.String(), nonFormatter)
	}
}

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
