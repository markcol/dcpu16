package dcpu16

import (
	"math"
	"sync"
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
	// The following registers are exported in the Register call
	O
	SP
	PC
	TICK
	regSize = iota // number of exported registers
)

// DCPU16 is a single virtual CPU that conforms to the 0x10c.com dcpu16 spec.
// The CPU can be run in a separate goroutine. The state access functions
// (Read, Write, Registers, etc.) will only be executed at instruction boundaries,
// ensuring that the state returned is consistent and atomic with respect to
// the virtual CPU.
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
	r[PC] = c.pc
	r[SP] = c.sp
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
	var opcode uint16

	// hold lock during entire instruction cycle
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// read the opcode
	opcode = c.nextWord()

	// check for extended opcode
	if (opcode & 0x0f) == 0 {
		c.extended(opcode)
	} else {
		c.standard(opcode)
	}

	// calculate the cycle count. NB: count is 1 less then spec, since nextWork
	// adds 1 to count during the opcode fetch.
	switch opcode & 0x0f {
	case 0x5, 0x6:
		c.tick += 2
	case 0x1, 0x9, 0xa, 0xb:
		break
	default:
		c.tick++
	}
}

// standard executes single a (non-extended) instruction opcode.
func (c *DCPU16) standard(opcode uint16) {
	var (
		a, b       *uint16
		aval, bval uint16
	)

	// fetch and evaluate a, then b
	a = c.lea((opcode&0x3f0)>>4, &aval)
	b = c.lea((opcode&0xfc00)>>10, &bval)

	// "If any instruction tries to assign a literal value, the assignment
	// fails silently. Other than that, the instruction behaves as normal."
	if a == &aval && (opcode >= 0x01 && opcode <= 0x0b) {
		return
	}

	switch opcode & 0x0f {
	case 0x1: // SET a, b - sets a to b
		*a = *b
	case 0x2: // ADD a, b - sets a to a+b, sets O to 0x0001 if there's an overflow, 0x0 otherwise
		v := uint32(*a) + uint32(*b)
		if v > math.MaxUint16 {
			c.overflow = 1
		} else {
			c.overflow = 0
		}
		*a = uint16(v)
	case 0x3: // SUB a, b - sets a to a-b, sets O to 0xffff if there's an underflow, 0x0 otherwise
		v := int32(*a) - int32(*b)
		if v < 0 {
			c.overflow = 0xffff
		} else {
			c.overflow = 0
		}
		*a = uint16(v)
	case 0x4: // MUL a, b - sets a to a*b, sets O to ((a*b)>>16)&0xffff
		v := int32(*a) * int32(*b)
		c.overflow = uint16(v >> 16)
		*a = uint16(v)
	case 0x5: // DIV a, b - sets a to a/b, sets O to ((a<<16)/b)&0xffff. if b==0, sets a and O to 0 instead.
		if *b != 0 {
			v := int32(*a) / int32(*b)
			c.overflow = uint16(v >> 16)
			*a = uint16(v)
		} else {
			*a = 0
			c.overflow = 0
		}
	case 0x6: // MOD a, b - sets a to a%b. if b==0, sets a to 0 instead.
		if *b == 0 {
			*a = 0
		} else {
			*a %= *b
		}
	case 0x7: // SHL a, b - sets a to a<<b, sets O to ((a<<b)>>16)&0xffff
		c.overflow = uint16(((uint32(*a) << *b) >> 16))
		*a <<= *b
	case 0x8: // SHR a, b - sets a to a>>b, sets O to ((a<<16)>>b)&0xffff
		c.overflow = uint16(((uint32(*a) << 16) >> *b))
		*a >>= *b
	case 0x9: // AND a, b - sets a to a&b
		*a &= *b
	case 0xa: // BOR a, b - sets a to a|b
		*a |= *b
	case 0xb: // XOR a, b - sets a to a^b
		*a ^= *b
	case 0xc: // IFE a, b - performs next instruction only if a==b
		if !(*a == *b) {
			c.nextWord()
		}
	case 0xd: // IFN a, b - performs next instruction only if a!=b
		if !(*a != *b) {
			c.nextWord()
		}
	case 0xe: // IFG a, b - performs next instruction only if a>b
		if !(*a > *b) {
			c.nextWord()
		}
	case 0xf: // IFB a, b - performs next instruction only if (a&b)!=0
		if !((*a & *b) != 0) {
			c.nextWord()
		}
	}
}

// extnded executes a single extended instruction opcode.
func (c *DCPU16) extended(opcode uint16) {
	var (
		a    *uint16
		aval uint16
	)

	a = c.lea((opcode&0xfc00)>>10, &aval)
	switch (opcode & 0x3f) >> 4 {
	case 0x1: // JSR a
		c.pushValue(c.pc)
		c.pc = *a
	default:
		panic("Invalid extended opcode")
	}
}

// lea returns the address of the value given by the addr operand. cval
// provides a pointer to the location to store constant values.
//
// Note this function returns a host pointer to guest memory, register, or
// a host-provided constant buffer.
func (c *DCPU16) lea(addr uint16, cval *uint16) *uint16 {
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
	case addr <= 0x3f: // literal value 0x00-0x1f (literal)
		*cval = addr - 0x20
		return cval
	}
	panic("Invalid address specification")
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
	v = &c.memory[c.sp]
	return
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
