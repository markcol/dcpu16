package cpu

import (
	"math"
	"sync"
	"time"
)

const (
	RAMSIZE              = 0x10000                 // 65535 words of RAM
	LASTADDR             = 0xffff                  // Last valid address
	CYCLERATE            = 1000                    // instructions/second
	INSTRUCTION_DURATION = time.Second / CYCLERATE // duration of an instruction
)

// OPCODE constants
const (
	EXTENDED = iota
	SET
	ADD
	SUB
	MUL
	DIV
	MOD
	SHL
	SHR
	AND
	BOR
	XOR
	IFE
	IFN
	IFG
	IFB
)

// Extended OPCODE constants
const (
	_ = iota
	JSR
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
	// The following registers are exported by the Register call but are not
	// really registers as defined by the specification. (e.g., they are not
	// used by register-relative addressing, etc.
	O
	SP
	PC
	TICK
	regSize = iota // number of exported registers
)

// Various constants to simplify coding
const (
	OPC_MASK     = 0x000f // normal instruction opcode mask (lower 4 bits of opcode)
	OP1_MASK     = 0x03F0 // first operand mask (a in normal instruction)
	OP2_MASK     = 0xfc00 // second operand mask (b in normal, a in extended instruction)
	OPERAND_MASK = 0x3f   // lower 6-bits of word

)

// DCPU16 is a single virtual CPU that conforms to the 0x10c.com dcpu16 spec.
// The CPU can be run in a separate goroutine. The state access functions
// (Read, Write, Registers, etc.) will only be executed at instruction boundaries,
// ensuring that the state returned is consistent and atomic with respect to
// the virtual CPU instruction cycle.
type DCPU16 struct {
	register [8]uint16
	memory   [RAMSIZE]uint16
	overflow uint16
	sp       uint16
	pc       uint16
	tick     uint16
	mutex    sync.Mutex
}

// Write writes the words from the slice data into memory starting at the
// address in addr. Any existing data will be overwritten.
// If addr + len(data) > MEMSIZE, only MEMSIZE-addr+1 words will be copied.
func (c *DCPU16) Write(addr uint16, data []uint16) {
	// wait for an instruction boundary
	c.mutex.Lock()
	defer c.mutex.Unlock()

	copy(c.memory[addr:], data)
}

// Read reads (at most) len words from memory starting at the given address and
// returns them to the caller. The number of words returned may be less than
// requested if address+len exeeds addressable memory.
func (c *DCPU16) Read(addr uint16, l int) []uint16 {
	// wait for an instruction boundary
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if int(addr)+l > LASTADDR {
		l = LASTADDR - int(addr) + 1
	}
	d := make([]uint16, l)
	copy(d, c.memory[addr:])
	return d
}

// Registers returns a slice of words that contains the value of the current
// CPU registers. The registers are stored in the following order: a, b, c, x,
// y, z, i, j, o, sp, pc, tick.
func (c *DCPU16) Registers() []uint16 {
	// wait for an instruction boundary
	c.mutex.Lock()
	defer c.mutex.Unlock()

	r := make([]uint16, regSize)
	copy(r, c.register[:])
	r[O] = c.overflow
	r[SP] = c.sp
	r[PC] = c.pc
	r[TICK] = c.tick
	return r
}

// Step executes a single instruction and returns to the caller.
func (c *DCPU16) Step() {
	c.step()
}

// Run executes instructions endlessly.
func (c *DCPU16) Run() {
	for true {
		c.step()
	}
}

// step executes a single machine instruction at [pc], updating all registers,
// memory, and cycle counts.
func (c *DCPU16) step() {
	var (
		opcode uint16
		wait   time.Duration
	)

	// hold lock during entire instruction cycle
	c.mutex.Lock()
	defer c.mutex.Unlock()

	start := time.Now()
	oldtick := c.tick

	opcode = c.nextWord()
	c.execute(opcode)

	// Calculate the cycle count. Note: nextWord increments tick count.
	switch opcode & OPC_MASK {
	case DIV, MOD:
		c.tick += 2
	case SET, AND, BOR, XOR:
		break
	default:
		c.tick++
	}
	if c.tick < oldtick {
		wait = time.Duration(c.tick + (0xffff - oldtick) + 1)
	} else {
		wait = time.Duration(c.tick - oldtick)
	}

	// Calculate the amount of time left before end of instruction cycle, and
	// sleep if there is time left.
	end := time.Now()
	wait = wait*INSTRUCTION_DURATION - end.Sub(start)
	if wait > 0 {
		time.Sleep(wait)
	}
}

// execute executes single a standard or extended instruction.
//
// The bit-level layout of a basic instructon (with lsb last) has the form:
// bbbbbbaaaaaaoooo. Where o, a, b are opcode, a-value, b-value respectively.
func (c *DCPU16) execute(opcode uint16) {
	var (
		a, b       *uint16
		aval, bval uint16
	)

	//fetch and evaluate a, then b
	a = c.lea((opcode&OP1_MASK)>>4, &aval)
	b = c.lea((opcode&OP2_MASK)>>10, &bval)

	// "If any instruction tries to assign a literal value, the assignment
	// fails silently. Other than that, the instruction behaves as normal."
	if a == &aval && (opcode >= 0x01 && opcode <= 0x0b) {
		return
	}

	switch opcode & OPC_MASK {
	case EXTENDED: // extended opcode
		switch *a { // *a = extended opcode, *b = operand
		case JSR: // push current PC onto stack, set PC = a
			c.pushValue(c.pc)
			c.pc = *b
		default:
			// panic("Invalid extended opcode")
		}

	case SET: // sets a to b
		*a = *b
	case ADD: // sets a to a+b, sets O to 0x0001 if there's an overflow, 0x0 otherwise
		v := uint32(*a) + uint32(*b)
		if v > math.MaxUint16 {
			c.overflow = 1
		} else {
			c.overflow = 0
		}
		*a = uint16(v)
	case SUB: // sets a to a-b, sets O to 0xffff if there's an underflow, 0x0 otherwise
		v := int32(*a) - int32(*b)
		if v < 0 {
			c.overflow = 0xffff
		} else {
			c.overflow = 0
		}
		*a = uint16(v)
	case MUL: // sets a to a*b, sets O to ((a*b)>>16)&0xffff
		v := int32(*a) * int32(*b)
		c.overflow = uint16(v >> 16)
		*a = uint16(v)
	case DIV: // sets a to a/b, sets O to ((a<<16)/b)&0xffff. if b==0, sets a and O to 0 instead.
		if *b != 0 {
			v := int32(*a) / int32(*b)
			c.overflow = uint16(v >> 16)
			*a = uint16(v)
		} else {
			*a = 0
			c.overflow = 0
		}
	case MOD: // sets a to a%b. if b==0, sets a to 0 instead.
		if *b == 0 {
			*a = 0
		} else {
			*a %= *b
		}
	case SHL: // sets a to a<<b, sets O to ((a<<b)>>16)&0xffff
		c.overflow = uint16(((uint32(*a) << *b) >> 16))
		*a <<= *b
	case SHR: // sets a to a>>b, sets O to ((a<<16)>>b)&0xffff
		c.overflow = uint16(((uint32(*a) << 16) >> *b))
		*a >>= *b
	case AND: // sets a to a&b
		*a &= *b
	case BOR: // sets a to a|b
		*a |= *b
	case XOR: // sets a to a^b
		*a ^= *b
	case IFE: // performs next instruction only if a==b
		if !(*a == *b) {
			c.nextWord()
		}
	case IFN: // performs next instruction only if a!=b
		if !(*a != *b) {
			c.nextWord()
		}
	case IFG: // performs next instruction only if a>b
		if !(*a > *b) {
			c.nextWord()
		}
	case IFB: // performs next instruction only if (a&b)!=0
		if !((*a & *b) != 0) {
			c.nextWord()
		}
	}
}

// lea (Load Effective Address) returns the address of the value given by the addr operand. cval
// provides a pointer to the location to store constant values.
//
// Note this function returns a host pointer to guest memory, register, or
// a host-provided constant buffer.
func (c *DCPU16) lea(addr uint16, cval *uint16) *uint16 {
	addr &= OPERAND_MASK
	switch {
	case addr <= 0x07: // register
		return &c.register[addr]
	case addr <= 0x0f: // [register]
		return &c.memory[c.register[addr-0x08]]
	case addr <= 0x17: // [next word + register]
		c.tick++
		return &c.memory[c.nextWord()+c.register[addr-0x10]]
	case addr == 0x18: // POP
		return c.pop()
	case addr == 0x19: // PEEK
		return &c.memory[c.sp]
	case addr == 0x1a: // PUSH
		return c.push()
	case addr == 0x1b: // SP
		return &c.sp
	case addr == 0x1c: // PC
		return &c.pc
	case addr == 0x1d: // O (overflow register)
		return &c.overflow
	case addr == 0x1e: // [next word]
		c.tick++
		return &c.memory[c.nextWord()]
	case addr == 0x1f: // next word (literal)
		c.tick++
		*cval = c.nextWord()
		return cval
	case addr <= OPERAND_MASK: // literal value 0x00-0x1f (literal)
		*cval = addr - 0x20
		return cval
	}
	// Should never happen, since value is limited at entry.
	panic("Invalid addressing mode.")
}

// nextWord returns the value of the memory at [pc] and increments the pc.
func (c *DCPU16) nextWord() (v uint16) {
	v = c.memory[c.pc]
	c.pc++
	c.tick++
	return
}

// push returns the value &[--sp]
// Note: returns a host pointer to the guest memory.
func (c *DCPU16) push() (v *uint16) {
	c.sp--
	return &c.memory[c.sp]
}

// pushValue pushes the word val onto the stack.
func (c *DCPU16) pushValue(val uint16) {
	c.sp--
	c.memory[c.sp] = val
}

// pop returns the value &[sp++]
// Note: returns a host pointer to the guest memory.
func (c *DCPU16) pop() (v *uint16) {
	v = &c.memory[c.sp]
	c.sp++
	return
}
