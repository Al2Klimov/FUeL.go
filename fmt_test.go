package fuel

import (
	"bytes"
	"io"
	"testing"
)

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

type failWrite struct {
	err error
}

var _ io.Writer = failWrite{}

func (fw failWrite) Write([]byte) (int, error) {
	return 0, fw.err
}
