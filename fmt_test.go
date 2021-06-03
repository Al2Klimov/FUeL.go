package fuel

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func TestFmtStateToString(t *testing.T) {
	assertFmtStateToString(t, &Formatable{}, "")
	assertFmtStateToString(t, &Formatable{Flags: map[int]struct{}{'+': {}}}, "+")
	assertFmtStateToString(t, &Formatable{Wid: 0, HasWid: true}, "0")
	assertFmtStateToString(t, &Formatable{Prec: 0, HasPrec: true}, ".0")

	assertFmtStateToString(t, &Formatable{
		Wid:     1,
		Prec:    2,
		Flags:   map[int]struct{}{'0': {}},
		HasWid:  true,
		HasPrec: true,
	}, "01.2")
}

func assertFmtStateToString(t *testing.T, in *Formatable, out string) {
	t.Helper()

	in.Output = disallowWrite{}

	if actual := FmtStateToString(in); actual != out {
		t.Errorf("FmtStateToString(%#v): got %#v, expected %#v", in, actual, out)
	}
}

func TestFormatNonFormatter(t *testing.T) {
	assertFormatNonFormatter(t, &Formatable{
		Flags: map[int]struct{}{'+': {}},
	}, 'd', testFormatter{"x"}, "self=x, state=+, verb=d")

	assertFormatNonFormatter(t, &Formatable{
		Wid:    2,
		Flags:  map[int]struct{}{'0': {}},
		HasWid: true,
	}, 'd', 1, "01")
}

func assertFormatNonFormatter(t *testing.T, fs *Formatable, verb rune, nonFormatter interface{}, out string) {
	t.Helper()

	buf := &bytes.Buffer{}
	fs.Output = buf

	FormatNonFormatter(fs, verb, nonFormatter)

	if actual := buf.String(); actual != out {
		t.Errorf("FormatNonFormatter(%#v, %#v, %#v): got %#v, expected %#v", fs, verb, nonFormatter, actual, out)
	}
}

func TestFormatable_Write(t *testing.T) {
	var f Formatable
	buf := &bytes.Buffer{}
	const dummy = "42"

	f.Output = buf

	if n, err := f.Write([]byte(dummy)); err == nil {
		if n == len(dummy) {
			if actual := buf.String(); actual != dummy {
				t.Errorf("Formatable#Write([]byte(%#v)): written %#v, expected %#v", dummy, actual, dummy)
			}

			if f.Error != nil {
				t.Errorf("Formatable#Write([]byte(%#v)): Formatable#Error %#v, expected nil", dummy, f.Error)
			}
		} else {
			t.Errorf("Formatable#Write([]byte(%#v)): written %#v, expected %#v", dummy, n, len(dummy))
		}
	} else {
		t.Errorf("Formatable#Write([]byte(%#v)): error %#v, expected nil", dummy, err)
	}

	f.Output = failWrite{io.EOF}

	if n, err := f.Write([]byte(dummy)); err != nil && err.Error() == io.EOF.Error() {
		if _, ok := err.(ErrorWithStack); !ok {
			t.Errorf("Formatable#Write([]byte(%#v)): plain error, expected ErrorWithStack", dummy)
		}

		if n == 0 {
			if f.Error.Error() != io.EOF.Error() {
				t.Errorf("Formatable#Write([]byte(%#v)): Formatable#Error %#v, expected io.EOF", dummy, f.Error)
			}
		} else {
			t.Errorf("Formatable#Write([]byte(%#v)): written %#v, expected 0", dummy, n)
		}
	} else {
		t.Errorf("Formatable#Write([]byte(%#v)): error %#v, expected io.EOF", dummy, err)
	}

	f.Output = failWrite{io.ErrClosedPipe}

	if n, err := f.Write([]byte(dummy)); err != nil && err.Error() == io.ErrClosedPipe.Error() {
		if _, ok := err.(ErrorWithStack); !ok {
			t.Errorf("Formatable#Write([]byte(%#v)): plain error, expected ErrorWithStack", dummy)
		}

		if n == 0 {
			if f.Error.Error() != io.EOF.Error() {
				t.Errorf("Formatable#Write([]byte(%#v)): Formatable#Error %#v, expected io.EOF", dummy, f.Error)
			}
		} else {
			t.Errorf("Formatable#Write([]byte(%#v)): written %#v, expected 0", dummy, n)
		}
	} else {
		t.Errorf("Formatable#Write([]byte(%#v)): error %#v, expected io.ErrClosedPipe", dummy, err)
	}
}

type disallowWrite struct {
}

var _ io.Writer = disallowWrite{}

func (dw disallowWrite) Write([]byte) (int, error) {
	panic("don't call me")
}

type testFormatter struct {
	id string
}

var _ fmt.Formatter = testFormatter{}

func (tf testFormatter) Format(fs fmt.State, verb rune) {
	fmt.Fprintf(fs, "self=%s, state=%s, verb=%c", tf.id, FmtStateToString(fs), verb)
}

type failWrite struct {
	err error
}

var _ io.Writer = failWrite{}

func (fw failWrite) Write([]byte) (int, error) {
	return 0, fw.err
}
