package dcpu16

import (
	"math"
)

const (
	RAMSIZE  = 0x10000 // 65535 words of RAM
	LASTADDR = 0xffff  // Last valid address
)

// Register offsets
const (
	A = iota
	B
	C
	X
	Y
	Z
	I
	J
	O       // overflow
	PC      // program counter
	SP      // stack pointern
	TICK    // clock ticks (cycle counter)
	regSize // number of registers (for sizing register slice)
)

var (
	register = make([]uint16, regSize)
	memory   = make([]uint16, RAMSIZE)
	sp       = &register[SP]
	pc       = &register[PC]
	cycles   = &register[TICK]
)

// Write writes the words from the slice data into memory starting at the
// address in addr. Any existing data will be overwritten.
//
// If addr + len(data) > MEMSIZE, only MEMSIZE-addr+1 words will be copied.
func Write(addr uint16, data []uint16) {
	copy(memory[addr:], data)
}

// Read reads (at most) len words from memory starting at the given address and
// returns them to the caller. The number of words returned may be less than
// requested if address+len exeeds addressable memory.
func Read(addr uint16, l int) []uint16 {
	if int(addr)+l > LASTADDR {
		l = LASTADDR - int(addr) + 1
	}
	d := make([]uint16, l)
	copy(d, memory[addr:])
	return d
}

// Registers returns a slice of words that contains the value of the current
// CPU registers. The registers are stored in the following order: a, b, c, x,
// y, z, i, j, o, pc, sp, tick.
func Registers() []uint16 {
	r := make([]uint16, regSize)
	copy(r, register)
	return r
}

// Reset sets all registers and memory to zero
func Reset() {
	for i := range register {
		register[i] = 0
	}
	for i := range memory {
		memory[i] = 0
	}
}

// Step executes a single instruction and returns to the caller.
func Step() {
	step()
}

// Run executes instructions endlessly.
func Run() {
	for true {
		step()
	}
}

// step executes a single machine instruction at [pc], updating all registers,
// memory, and cycle counts.
func step() {
	var opcode uint16

	// read the opcode
	opcode = nextWord()

	// check for extended opcode
	if (opcode & 0x0f) == 0 {
		extended(opcode)
	} else {
		standard(opcode)
	}

	// calculate the cycle count. NB: count is 1 less then spec, since nextWork
	// adds 1 to count during the opcode fetch.
	switch opcode & 0x0f {
	case 0x5, 0x6:
		(*cycles) += 2
	case 0x1, 0x9, 0xa, 0xb:
		break
	default:
		(*cycles) += 1
	}
}

// standard executes single a (non-extended) instruction opcode.
func standard(opcode uint16) {
	var (
		a, b       *uint16
		aval, bval uint16
		v          int
	)

	// fetch and evaluate a, then b
	a = lea((opcode&0x3f0)>>4, &aval)
	b = lea((opcode&0xfc00)>>10, &bval)

	// "If any instruction tries to assign a literal value, the assignment
	// fails silently. Other than that, the instruction behaves as normal."
	if a == &aval && (opcode >= 0x01 && opcode <= 0x0b) {
		return
	}

	switch opcode & 0x0f {
	case 0x1: // SET a, b - sets a to b
		*a = *b
	case 0x2: // ADD a, b - sets a to a+b, sets O to 0x0001 if there's an overflow, 0x0 otherwise
		v = int(*a) + int(*b)
		if v > math.MaxInt16 {
			register[O] = 1
		} else {
			register[O] = 0
		}
		*a = uint16(v)
	case 0x3: // SUB a, b - sets a to a-b, sets O to 0xffff if there's an underflow, 0x0 otherwise
		v = int(*a) - int(*b)
		if v < math.MinInt16 {
			register[O] = 1
		} else {
			register[O] = 0
		}
		*a = uint16(v)
	case 0x4: // MUL a, b - sets a to a*b, sets O to ((a*b)>>16)&0xffff
		register[O] = ((*a * *b) >> 16) & 0xffff
		*a *= *b
	case 0x5: // DIV a, b - sets a to a/b, sets O to ((a<<16)/b)&0xffff. if b==0, sets a and O to 0 instead.
		if *b == 0 {
			*a = 0
			register[O] = 0
		} else {
			register[O] = ((*a << 16) / *b) & 0xffff
			*a /= *b
		}
	case 0x6: // MOD a, b - sets a to a%b. if b==0, sets a to 0 instead.
		if *b == 0 {
			*a = 0
		} else {
			*a %= *b
		}
	case 0x7: // SHL a, b - sets a to a<<b, sets O to ((a<<b)>>16)&0xffff
		*a <<= *b
		register[O] = ((*a << *b) >> 16) & 0x0ffff
	case 0x8: // SHR a, b - sets a to a>>b, sets O to ((a<<16)>>b)&0xffff
		*a >>= *b
		register[O] = ((*a << *b) >> 16) & 0x0ffff
	case 0x9: // AND a, b - sets a to a&b
		*a &= *b
	case 0xa: // BOR a, b - sets a to a|b
		*a |= *b
	case 0xb: // XOR a, b - sets a to a^b
		*a ^= *b
	case 0xc: // IFE a, b - performs next instruction only if a==b
		if !(*a == *b) {
			nextWord()
		}
	case 0xd: // IFN a, b - performs next instruction only if a!=b
		if !(*a != *b) {
			nextWord()
		}
	case 0xe: // IFG a, b - performs next instruction only if a>b
		if !(*a > *b) {
			nextWord()
		}
	case 0xf: // IFB a, b - performs next instruction only if (a&b)!=0
		if !((*a & *b) != 0) {
			nextWord()
		}
	}
}

// extnded executes a single extended instruction opcode.
func extended(opcode uint16) {
	var (
		a    *uint16
		aval uint16
	)

	a = lea((opcode&0xfc00)>>10, &aval)
	switch (opcode & 0x3f) >> 4 {
	case 0x1: // JSR a
		pushValue(*pc)
		*pc = *a
	default:
		panic("Invalid extended opcode")
	}
}

// lea returns the address of the value given by the addr operand. cval
// provides a pointer to the location to store constant values.
//
// Note this function returns a host pointer to guest memory, register, or
// a host-provided constant buffer.
func lea(addr uint16, cval *uint16) *uint16 {
	switch {
	case addr <= 0x07: // register
		return &register[addr]
	case addr <= 0x0f: // [register]
		return &memory[register[addr-0x08]]
	case addr <= 0x17: // [next word + register]
		(*cycles)++
		return &memory[nextWord()+register[addr-0x10]]
	case addr == 0x18: // POP
		return pop()
	case addr == 0x19: // PEEK
		return &memory[*sp]
	case addr == 0x1a: // PUSH
		return push()
	case addr == 0x1b: // SP
		return sp
	case addr == 0x1c: // PC
		return pc
	case addr == 0x1d: // O (overflow register)
		return &register[O]
	case addr == 0x1e: // [next word]
		(*cycles)++
		return &memory[nextWord()]
	case addr == 0x1f: // next word (literal)
		(*cycles)++
		*cval = nextWord()
		return cval
	case addr <= 0x3f: // literal value 0x00-0x1f (literal)
		*cval = addr - 0x20
		return cval
	}
	panic("Invalid address specification")
}

// nextWord returns the value of the memory at [pc] and increments the pc.
func nextWord() (v uint16) {
	v = memory[*pc]
	(*pc)++
	(*cycles)++
	return
}

// push returns the value &[--sp]
// Note: returns a host pointer to the guest memory.
func push() (v *uint16) {
	(*sp)--
	v = &memory[*sp]
	return
}

// pushValue pushes the word val onto the stack.
func pushValue(val uint16) {
	(*sp)--
	memory[*sp] = val
}

// pop returns the value &[sp++]
// Note: returns a host pointer to the guest memory.
func pop() (v *uint16) {
	v = &memory[*sp]
	(*sp)++
	return
}
