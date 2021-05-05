package objects

import (
	"fmt"
	"io"

	"github.com/wrgl/core/pkg/encoding"
)

func writeLine(w io.Writer, label string, b []byte) (n int, err error) {
	for _, sl := range [][]byte{
		[]byte(label), {' '}, b, {'\n'},
	} {
		n, err = w.Write(sl)
		if err != nil {
			return
		}
	}
	return
}

func consumeStr(p *encoding.Parser, s string) error {
	b, err := p.NextBytes(len(s))
	if err != nil {
		return err
	}
	if string(b) != s {
		return p.ParseError("expected string %q, received %q", s, string(b))
	}
	return nil
}

func readLine(p *encoding.Parser, label string, f encoding.DecodeFunc) error {
	err := consumeStr(p, fmt.Sprintf("%s ", label))
	if err == io.EOF {
		return io.EOF
	}
	if err != nil {
		return fmt.Errorf("error reading label %q: %v", label, err)
	}
	err = f(p)
	if err != nil {
		return fmt.Errorf("error reading label %q: %v", label, err)
	}
	err = consumeStr(p, "\n")
	if err != nil {
		return fmt.Errorf("error reading label %q: %v", label, err)
	}
	return nil
}
