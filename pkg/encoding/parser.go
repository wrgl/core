// SPDX-License-Identifier: Apache-2.0
// Copyright © 2021 Wrangle Ltd

package encoding

import (
	"fmt"
	"io"

	"github.com/wrgl/core/pkg/misc"
)

// Parser keep track of read position and read into a buffer
type Parser struct {
	pos int
	buf Bufferer
	r   io.Reader
}

func NewParser(r io.Reader) *Parser {
	return &Parser{
		r:   r,
		buf: misc.NewBuffer(nil),
	}
}

func (r *Parser) ParseError(format string, a ...interface{}) error {
	return fmt.Errorf("parse error at pos=%d: %s", r.pos, fmt.Sprintf(format, a...))
}

func (r *Parser) NextBytes(n int) ([]byte, error) {
	b := r.buf.Buffer(n)
	err := r.ReadBytes(b)
	return b, err
}

func (r *Parser) ReadBytes(b []byte) error {
	n, err := r.r.Read(b)
	r.pos += n
	return err
}
