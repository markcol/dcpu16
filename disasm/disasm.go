package disasm

import (
	"fmt"
	"io"
	"github.com/markcol/dcpu16/cpu"
)

var (
	register = []string{"A", "B", "C", "X", "Y", "Z", "I", "J"}
	opcodes  = map[int]string{1: "SET", 2: "ADD", 3: "SUB", 4: "MUL", 5: "DIV", 6: "MOD", 7: "SHL", 8: "SHR", 9: "AND", 10: "BOR", 11: "XOR", 12: "IFE", 13: "IFN", 14: "IFG", 15: "IFB"}
)

type wordReader struct {
	m []uint16
	i int
}

func NewWordReader(m []uint16) WordReader { return &wordReader{m, 0} }

func (r *wordReader) ReadWord() (w uint16, err error) {
	if r.i >= len(r.m) {
		return 0, io.EOF
	}
	w = r.m[r.i]
	r.i++
	return
}

type WordReader interface {
	ReadWord() (w uint16, err error)
}

func disasm(addr uint16, r WordReader, w io.Writer) {
	var a, b string
	var err error
	var v uint16

	for true {
		oldAddr := addr
		v, err = r.ReadWord()
		addr++
		if err != nil {
			break
		}
		op := v & 0x0f
		if op >= 0x01 && op <= 0x0f {
			a, addr, err = addrMode(v>>4&0x3f, addr, r)
			if err != nil {
				break
			}
			b, addr, err = addrMode(v>>10&0x3f, addr, r)
			if err != nil {
				break
			}
			w.Write([]byte(fmt.Sprintf("0x%04x:\t\t%s\t%s, %s\n", oldAddr, opcodes[int(op)], a, b)))
		} else if op == 0 && (v&0x3f) == 0x10 {
			a, addr, err = addrMode(v>>10&0x3f, addr, r)
			if err != nil {
				break
			}
			w.Write([]byte(fmt.Sprintf("0x%04x:\t\tJSR\t%s\n", oldAddr, a)))
		} else {
			w.Write([]byte(fmt.Sprintf("0x%04x:\t%04x\n", oldAddr, v)))
		}
	}
	w.Write([]byte("\n"))
}

func addrMode(opcode uint16, a uint16, r WordReader) (s string, addr uint16, err error) {
	addr = a
	switch {
	case opcode <= 0x07:
		return register[opcode], addr, nil
	case opcode <= 0x0f:
		return fmt.Sprintf("[%s]", register[opcode-0x08]), addr, nil
	case opcode <= 0x17:
		v, err := r.ReadWord()
		addr++
		return fmt.Sprintf("[0x%x+%s]", v, register[opcode-0x10]), addr, err
	case opcode <= 0x18:
		return "POP", addr, nil
	case opcode == 0x19:
		return "PEEK", addr, nil
	case opcode == 0x1a:
		return "PUSH", addr, nil
	case opcode == 0x1b:
		return "SP", addr, nil
	case opcode == 0x1c:
		return "PC", addr, nil
	case opcode == 0x1d:
		return "O", addr, nil
	case opcode == 0x1e:
		v, err := r.ReadWord()
		addr++
		return fmt.Sprintf("[0x%x]", v), addr, err
	case opcode == 0x1f:
		v, err := r.ReadWord()
		addr++
		return fmt.Sprintf("0x%x", v), addr, err
	case opcode >= 0x020 && opcode <= 0x3f:
		return fmt.Sprintf("0x%02x", opcode-0x20), addr, nil
	}
	return "Unknown", addr, nil
}
