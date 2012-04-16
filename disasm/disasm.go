package disasm

import (
	"github.com/markcol/dcpu16/cpu"
	"io"
)

var (
	registers = []string{"A", "B", "C", "X", "Y", "Z", "I", "J"}
	opcodes   = map[int]string{1: "SET", 2: "ADD", 3: "SUB", 4: "MUL", 5: "DIV", 6: "MOD", 7: "SHL", 8: "SHR", 9: "AND", 10: "BOR", 11: "XOR", 12: "IFE", 13: "IFN", 14: "IFG", 15: "IFB"}
)

type wordReader struct {
	m []uint16
	i int
}

func NewWordReader(m []uint15) { return &wordReader{m, 0} }

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

func disasm(r WordReader, w Writer) {
	i := 0
	for true {
		if v, err := r.ReadWord(); err != nil {
			return
		}
		op := v & 0x0f
		if op >= 0x01 && op <= 0x0f {
			a, err := addrMode(v>>4&0x3f, r)
			b, err := addrMode(v>>10&0x3f, r)
			w.Write(fmt.Sprint("%s %s, %s\n", opcodes[op], a, b))
		} else if op == 0 && (v&0x3f) == 0x10 {
			a, err := addrMode(opcode>>10&0x3f, r)
			w.Write("JSR %s\n", a)
		} else {
			w.Write("%04x\n", v)
		}
	}
	w.Write("\n")
}

func addrMode(opcode uint16, r WordReader) (string, err error) {
	switch opcode {
	case opcode <= 0x07:
		return register[opcode], nil
	case opcode <= 0x0f:
		return fmt.Sprintf("[%s]", register[opcode-0x07]), nil
	case opcode <= 0x17:
		v, err := r.ReadWord()
		return fmt.Sprintf("[%04x+%s]", v, register[opcode-0x07]), err
	case opcode <= 0x18:
		return "POP", nil
	case opcode == 0x19:
		return "PEEK", nil
	case opcode == 0x1a:
		return "PUSH", nil
	case opcode == 0x1b:
		return "SP", nil
	case opcode == 0x1c:
		return "PC", nil
	case opcode == 0x1d:
		return "O", nil
	case opcode == 0x1e:
		v, err := r.ReadWord()
		return fmt.Sprintf("[%04x]", v), err
	case opcode == 0x1f:
		v, err := r.ReadWord()
		return fmt.Sprintf("%04x", v), err
	case opcode > 0x020 && opcode <= 0x3f:
		return fmt.Sprintf("%02x", opcode-0x20), nil
	}
}
