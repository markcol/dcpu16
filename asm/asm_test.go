package asm

import (
	"testing"
)

func TestSimple(t *testing.T) {
	input := "; Try some basic stuff\n" +
		"              SET A, 0x30              ; 7c01 0030\n" +
		"              SET [0x1000], 0x20       ; 7de1 1000 0020\n" +
		"              SUB A, [0x1000]          ; 7803 1000\n" +
		"              IFN A, 0x10              ; c00d\n" +
		"              SET PC, crash            ; 7dc1 001a" +
		"\n" +
		"; Do a loopy thing\n" +
		"              SET I, 10                ; a861\n" +
		"              SET A, 0x2000            ; 7c01 2000\n" +
		":loop         SET [0x2000+I], [A]      ; 2161 2000\n" +
		"              SUB I, 1                 ; 8463\n" +
		"              IFN I, 0                 ; 806d\n" +
		"              SET PC, loop             ; 7dc1 000d\n" +
		"\n" +
		"; Call a subroutine\n" +
		"              SET X, 0x4               ; 9031\n" +
		"              JSR testsub              ; 7c10 0018 [*]\n" +
		"              SET PC, crash            ; 7dc1 001a [*]\n" +
		"\n" +
		":testsub      SHL X, 4                 ; 9037\n" +
		"              SET PC, POP              ; 61c1\n" +
		"\n" +
		"; Hang forever. X should now be 0x40 if everything went right.\n" +
		":crash        SET PC, crash            ; 7dc1 001a [*]\n"

	expect := []uint16{
		0x7c01, 0x0030, 0x7de1, 0x1000, 0x0020, 0x7803, 0x1000, 0xc00d,
		0x7dc1, 0x001a, 0xa861, 0x7c01, 0x2000, 0x2161, 0x2000, 0x8463,
		0x806d, 0x7dc1, 0x000d, 0x9031, 0x7c10, 0x0018, 0x7dc1, 0x001a,
		0x9037, 0x61c1, 0x7dc1, 0x001a,
	}

	_ = input
	_ = expect
}
