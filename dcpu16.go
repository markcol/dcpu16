package dcpu16

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
	MAX_INTQUEUE         = 256
)

// OPCODE constants
const (
	EXT = iota // Extended Opcode pseudo opcode
	SET
	ADD
	SUB
	MUL
	MLI
	DIV
	DVI
	MOD
	MDI
	AND
	BOR
	XOR
	SHR
	ASR
	SHL
	IFB
	IFC
	IFE
	IFN
	IFG
	IFA
	IFL
	IFU
	_
	_
	ADX
	SBX
	_
	_
	STI
	STD
)

// Extended OPCODE constants
const (
	_ = iota
	JSR
	_
	_
	_
	_
	_
	_
	INT
	IAG
	IAS
	RFI
	IAQ
	_
	_
	_
	HWN
	HWQ
	HWI
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
	PC             // Program Counter
	SP             // Stack Pointer
	EX             // Overflow Register
	IA             // Interrupt Address Register
	TICK           // tick counter
	IQ             // Interrupt Queuing flag
	regSize = iota // number of exported registers
)

// Various constants to simplify coding
const (
	OPCODE_MASK = 0x001f // normal instruction opcode mask
	ARGA_MASK   = 0xFC00 // first operand mask: a
	ARGB_MASK   = 0x03E0 // second operand mask: b
	ARGA_SHIFT  = 10
	ARGB_SHIFT  = 5
)

// DCPU16 is a single virtual CPU that conforms to the 0x10c.com dcpu16 spec.
// The CPU can be run in a separate goroutine. The state access functions
// (Read, Write, Registers, etc.) will only be executed at instruction boundaries,
// ensuring that the state returned is consistent and atomic with respect to
// the virtual CPU instruction cycle.
type DCPU16 struct {
	register    [8]uint16
	memory      [RAMSIZE]uint16
	pc          uint16
	sp          uint16
	ex          uint16
	ia          uint16
	tick        uint16
	intQueueing bool // true if interrupts are to be queued
	intQueue    []uint16
	tmpa        uint16
	tmpb        uint16
	mutex       sync.Mutex
}

func NewDCPU16() *DCPU16 {
	return &DCPU16{
		intQueue:    make([]uint16, 0, MAX_INTQUEUE),
		intQueueing: false,
	}
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
// requested if address + len exceeds addressable memory.
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

// Registers returns a slice of words with the values of the current CPU
// registers and pseudo-registers. The registers are stored in the following
// order: a, b, c, x, y, z, i, j, pc, sp, ex, ia, tick, iq.
func (c *DCPU16) Registers() []uint16 {
	// wait for an instruction boundary
	c.mutex.Lock()
	defer c.mutex.Unlock()

	r := make([]uint16, regSize)
	copy(r, c.register[:])
	r[PC] = c.pc
	r[SP] = c.sp
	r[EX] = c.ex
	r[IA] = c.ia
	r[TICK] = c.tick
	if c.intQueueing {
		r[IQ] = 1
	} else {
		r[IQ] = 0
	}
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
	var wait time.Duration

	// hold lock during entire instruction cycle
	c.mutex.Lock()
	defer c.mutex.Unlock()

	start := time.Now()
	oldtick := c.tick

	// execute the actual instruction
	c.execute()

	// process a software interrupt if queuing disabled and and one is queued
	if !c.intQueueing && len(c.intQueue) > 0 {
		a := c.intQueue[0]
		c.intQueue = c.intQueue[1:]
		if c.ia != 0 {
			c.intQueueing = true
			c.pushValue(c.pc)
			c.pushValue(c.register[A])
			c.pc = c.ia
			c.register[A] = a
		}
	}

	if c.tick < oldtick {
		// tick count rolled over through 0
		wait = time.Duration(c.tick + (math.MaxUint16 - oldtick) + 1)
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

// execute executes single a DCPU16 machine instruction.
//
// The bit-level layout of a basic instruction (with LSB on right) has the form:
// bbbbbbaaaaaaoooo. Where o, a, b are opcode, a-value, b-value respectively.
func (c *DCPU16) execute() {
	opcode := c.nextWord()
	a := c.lea((opcode&ARGA_MASK)>>ARGA_SHIFT, &c.tmpa)
	b := c.lea((opcode&ARGB_MASK)>>ARGB_SHIFT, &c.tmpb)

	if (b == &c.tmpb) && !(opcode >= IFB && opcode <= IFU) {
		// "If any instruction tries to assign a literal value, the assignment
		// fails silently. Other than that, the instruction behaves as normal."
		return
	}

	switch opcode & OPCODE_MASK {
	case EXT: // extended opcode
		// at entry, *a = extended opcode, *b = operand
		// reassign them for consistency with spec
		opcode = *a
		*a = *b
		switch opcode {
		case JSR: // push current PC onto stack, set PC = A
			c.pushValue(c.pc)
			c.pc = *a
			c.tick += 2
		case INT: // trigger a software interrupt with message A
			// Add interrupt to queue, process interrupt queue before next
			// instruction (if IAQ is zero).
			if len(c.intQueue) < MAX_INTQUEUE {
				c.intQueue = append(c.intQueue, *a)
			} else {
				panic("Interrupt queue exceeded: processor has caught fire!")
			}
			c.tick += 3
		case IAG: // sets A to IA
			*a = c.ia
		case IAS: // sets IA to A
			c.ia = *a
		case RFI: // return from interrupt: disable interrupt queuing, pop A, PC
			c.intQueueing = false
			c.register[A] = *c.pop()
			c.pc = *c.pop()
			c.tick += 2
		case IAQ: // if A is nonzero, interrupts will be queued, otherwise triggered
			c.intQueueing = (*a != 0)
			c.tick++
		case HWN: // sets A to number of connected hardware devices
			c.register[A] = 0
			c.tick++
		case HWQ: // returns device information about hardware A
			c.hardwareQuery(*a)
			c.tick += 3
		case HWI: // sends an interrupt to hardware A
			c.handleHardwareInterrupt(*a)
			c.tick += 3
		}

	case SET: // sets B to A
		*b = *a
	case ADD: // sets B to B+A, sets EX if there's an overflow, 0x0 otherwise
		v := uint32(*b) + uint32(*a)
		c.ex = uint16(v >> 16)
		*b = uint16(v)
		c.tick++
	case SUB: // sets B to B-A, sets EX if there's an underflow, 0x0 otherwise
		v := int32(*b) - int32(*a)
		c.ex = uint16(v >> 16)
		*b = uint16(v)
		c.tick++
	case MUL, MLI: // sets B to B*A, sets EX to ((B*A)>>16)&0xffff
		var v int32
		if opcode == MUL {
			// unsigned
			v = int32(uint32(*b) * uint32(*a))
		} else {
			// signed
			v = int32(*b) * int32(*a)
		}
		c.ex = uint16(v >> 16)
		*b = uint16(v)
		c.tick++
	case DIV, DVI: // sets B to B/A, sets EX ((B<<16)>>A)&0xffff
		var v int32
		if *a == 0 {
			*b = 0
			c.ex = 0
		} else {
			if opcode == DIV {
				// unsigned division
				v = int32(uint32(*b) / uint32(*a))
			} else {
				// signed division
				v = int32(*b) / int32(*a)
			}
			c.ex = uint16(v >> 16)
			*b = uint16(v)
		}
		c.tick += 2
	case MOD, MDI: // sets B to B%A. if A==0, sets B to 0 instead.
		if *a == 0 {
			*b = 0
		} else {
			if opcode == MOD {
				// signed
				*b %= *a
			} else {
				// unsigned
				*b = uint16(int16(*b) % int16(*a))
			}
		}
		c.tick += 2
	case AND: // sets B to B&A
		*b &= *a
	case BOR: // sets B to B|A
		*b |= *a
	case XOR: // sets B to B^A
		*b ^= *a
	case SHR: // sets B to B>>A, sets EX to ((B<<16)>>A)&0xffff
		c.ex = uint16(((uint32(*b) << 16) >> *a))
		*b >>= *a
	case ASR: // sets B to B>>A, sets EX to ((B<<16)>>>A)&0xffff (treats b as signed)
		c.ex = uint16(((int32(*b) << 16) >> *a))
		t := int16(*b)
		t >>= *a
		*b = uint16(t)
	case SHL: // sets B to B<<A, sets EX to ((B<<A)>>16)&0xffff
		c.ex = uint16(((uint32(*b) << *a) >> 16))
		*b <<= *a
	case IFB: // performs next instruction only if (B&A)!=0
		if !((*b & *a) != 0) {
			c.skipConditional()
		}
		c.tick++
	case IFC: // performs next instruction only if (B&A)==0
		if !((*b & *a) == 0) {
			c.skipConditional()
		}
		c.tick++
	case IFE: // performs next instruction only if B==A
		if !(*b == *a) {
			c.skipConditional()
		}
		c.tick++
	case IFN: // performs next instruction only if B!=A
		if !(*b != *a) {
			c.skipConditional()
		}
		c.tick++
	case IFG: // performs next instruction only if B > A
		if !(*b > *a) {
			c.skipConditional()
		}
		c.tick++
	case IFA: // performs next instruction only if B > A (signed)
		if !(int16(*b) > int16(*a)) {
			c.skipConditional()
		}
		c.tick++
	case IFL: // perform next instruction only if B < A
		if !(*b < *a) {
			c.skipConditional()
		}
		c.tick++
	case IFU: // perform next instruction only if B < A (signed)
		if !(int16(*b) < int16(*a)) {
			c.skipConditional()
		}
		c.tick++
	case ADX:
		v := int32(*b) + int32(*a) + int32(c.ex)
		if v > math.MaxInt16 {
			c.ex = 1
		} else {
			c.ex = 0
		}
		*b = uint16(v)
		c.tick += 2
	case SBX:
		v := int32(*b) - int32(*a) + int32(c.ex)
		if v < math.MinInt16 {
			c.ex = 0xffff
		} else {
			c.ex = 0
		}
		*b = uint16(v)
		c.tick += 2
	case STI, STD: // sets B to A, then increases / decreases I and J by 1
		*b = *a
		if opcode == STI {
			c.register[I]++
			c.register[J]++
		} else {
			c.register[I]--
			c.register[J]--
		}
		c.tick++
	}
	return
}

// lea (Load Effective Address) returns the address of the value given by the
// addr operand. tmp provides a pointer to the location to store constant
// values.
//
// Note this function returns a host pointer to guest memory, register, or
// constant buffer.
func (c *DCPU16) lea(addr uint16, tmp *uint16) *uint16 {
	switch {
	case addr <= 0x07: // register
		return &c.register[addr]
	case addr <= 0x0f: // [register]
		return &c.memory[c.register[addr-0x08]]
	case addr <= 0x17: // [next word + register]
		return &c.memory[c.nextWord()+c.register[addr-0x10]]
	case addr == 0x18: // POP (a) or PUSH (b)
		if tmp == &c.tmpa {
			return c.pop()
		}
		return c.push()
	case addr == 0x19: // PEEK
		return &c.memory[c.sp]
	case addr == 0x1a: // PICK n: [SP + next word]
		return &c.memory[c.sp+c.nextWord()]
	case addr == 0x1b: // SP
		return &c.sp
	case addr == 0x1c: // PC
		return &c.pc
	case addr == 0x1d: // EX
		return &c.ex
	case addr == 0x1e: // [next word]
		return &c.memory[c.nextWord()]
	case addr == 0x1f: // next word (literal)
		*tmp = c.nextWord()
		return tmp
	case addr <= 0x3f: // literal value 0xffff-0x1e (-1..30)
		*tmp = addr - 0x20 - 1
		return tmp
	}
	// will never be reached
	return nil
}

// skipConditional advances the PC to next word of memory. If the word being skipped
// is an IFx instruction, then skip two words (e.g., skip both branches of the
// IFx instruction), allowing for easy conditional chaining.
func (c *DCPU16) skipConditional() {
	op := c.nextWord()
	if op >= IFB && op <= IFU {
		c.nextWord()
	}
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

// The DCPU-16 supports up to 65535 connected hardware devices. These devices can
// be anything from additional storage, sensors, monitors or speakers.
// How to control the hardware is specified per hardware device, but the DCPU-16
// supports a standard enumeration method for detecting connected hardware via
// the HWN, HWQ and HWI instructions.
//
// Interrupts sent to hardware can't contain messages, can take additional cycles,
// and can read or modify any registers or memory addresses on the DCPU-16. This
// behavior changes per hardware device and is described in the hardware's
// documentation.
//
// Hardware must NOT start modifying registers or ram on the DCPU-16 before at
// least one HWI call has been made to the hardware.
//
// The DPCU-16 does not support hot swapping hardware. The behavior of connecting
// or disconnecting hardware while the DCPU-16 is running is undefined.

// hardwareQuery queries the hardware attached to the CPU and sets
// the A, B, C, X, Y registers to reflect the hardware device connected at
// port A. A+(B<<16) is a 32-bit word identifying the hardware ID. C is
// the hardware version. X+(Y<<16) is a 32-bit word identifying the
// manufacturer
func (c *DCPU16) hardwareQuery(hwindex uint16) {
	return
}

// handleHardwareInterrupt handles sending an interrupt to a hardware device
func (c *DCPU16) handleHardwareInterrupt(hwint uint16) {
	return
}
