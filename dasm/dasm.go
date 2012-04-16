package dasm

import (
	"io"
)

type WordWriter interface {
}

// Assemble assembles a DCPU16 assembly language program, reading the source
// file from r and writing the output to w.
func Assemble(r io.Reader, w WordWriter) (err error) {
	return nil
}
