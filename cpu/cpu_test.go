package cpu

import (
	"fmt"
	"testing"
)

const (
	PUSH = 0x18
	POP  = 0x18
	PEEK = 0x19
	PICK = 0x1a4
)

func TestWriteAndRead(t *testing.T) {
	c := new(DCPU16)
	c.Write(0, []uint16{0x7c01, 0x0030, 0x7de1})
	if c.memory[0] != 0x7c01 && c.memory[1] != 0x0030 && c.memory[2] != 0x7de1 {
		t.Errorf("Expected memory to equal written values, got: %v\n", c.memory[0:4])
	}
	m := c.Read(0, 4)
	if m[0] != 0x7c01 && m[1] != 0x0030 && m[2] != 0x7de1 {
		t.Errorf("Expected memory to equal written values, got: %v\n", m)
	}
}

func TestRegisters(t *testing.T) {
	c := new(DCPU16)
	// expect the registers to be zeroed
	e := make([]uint16, regSize)
	checkRegisters(e, c, t)
}

func TestSetA(t *testing.T) {
	c := new(DCPU16)
	c.memory[0] = makeOpcode(SET, 0x0, 0x1f) // SET A, 0x030
	c.memory[1] = 0x0030
	e := c.Registers()
	e[PC] = 2
	e[TICK] = 2
	e[A] = c.memory[1]
	c.step()
	if c.memory[0] != 0x7c01 {
		t.Errorf("Expected opcode %d, got %d\n", 0x7c01, c.memory[0])
	}

	checkRegisters(e, c, t)

}

func TestSetAllRegisters(t *testing.T) {
	c := new(DCPU16)
	e := c.Registers()
	for i := 0; i <= 7; i++ {
		c.memory[0] = makeOpcode(SET, i, 0x1f) // SET I, 0x0030
		c.memory[1] = 0x0030
		c.pc = 0
		e[PC] = 2
		e[TICK] += 2
		e[i] = c.memory[1]
		c.step()
		checkRegisters(e, c, t)
	}
}

func TestSetPC(t *testing.T) {
	c := new(DCPU16)
	c.memory[0] = makeOpcode(SET, 0x1c, 0x1f) // SET PC, 0x0030
	c.memory[1] = 0x0030
	e := c.Registers()
	e[PC] = c.memory[1]
	e[TICK] += 2
	c.step()
	checkRegisters(e, c, t)
}

func TestSetEX(t *testing.T) {
	c := new(DCPU16)
	c.memory[0] = makeOpcode(SET, 0x1d, 0x1f) // SET EX, 0x0030
	c.memory[1] = 0x0030
	e := c.Registers()
	e[EX] = c.memory[1]
	e[PC] = 2
	e[TICK] = 2
	c.step()
	checkRegisters(e, c, t)
}

func TestSetSP(t *testing.T) {
	c := new(DCPU16)
	c.memory[0] = makeOpcode(SET, 0x1b, 0x1f) // SET SP, 0x0030
	c.memory[1] = 0x0030
	e := c.Registers()
	e[SP] = c.memory[1]
	e[PC] = 2
	e[TICK] = 2
	c.step()
	checkRegisters(e, c, t)
}

func TestSetRegisterIndirect(t *testing.T) {
	c := new(DCPU16)
	c.memory[0] = makeOpcode(SET, 1, 0x0a) // SET B, [C]
	c.memory[1] = 0xabca
	c.register[C] = 1
	e := c.Registers()
	e[B] = c.memory[1]
	e[C] = 1
	e[PC] = 1
	e[TICK] = 1
	c.step()
	checkRegisters(e, c, t)
}

func TestSetRegisterIndirectOffset(t *testing.T) {
	c := new(DCPU16)
	c.memory[0] = makeOpcode(SET, 0x01, 0x10) // SET B, [0x0000+a]
	c.memory[1] = 0x0
	e := c.Registers()
	e[B] = c.memory[0]
	e[PC] = 2
	e[TICK] = 2
	c.step()
	checkRegisters(e, c, t)
}

func TestSetIndirect(t *testing.T) {
	c := new(DCPU16)
	c.memory[0] = makeOpcode(SET, 0x01, 0x1e) // SET B, [0x0002]
	c.memory[1] = 0x0002
	c.memory[2] = 0x7ce3
	e := c.Registers()
	e[B] = c.memory[2]
	e[PC] = 2
	e[TICK] = 2
	c.step()
	checkRegisters(e, c, t)
}

func TestSetAllShortLiterals(t *testing.T) {
	c := new(DCPU16)
	for i := 0; i <= 0x1f; i++ {
		c.pc = 0
		c.memory[0] = makeOpcode(SET, 0, 0x20+i) // SET A, i
		e := c.Registers()
		e[PC] = 1
		e[TICK] = c.tick + 1
		e[A] = uint16(i) - 1
		c.step()
		checkRegisters(e, c, t)
	}
}

func TestSetAssignLiteral(t *testing.T) {
	c := new(DCPU16)
	c.memory[0] = makeOpcode(SET, 0x1f, 0x3f) // SET 0x0030, 30
	c.memory[1] = 0x0030
	e := c.Registers()
	e[PC] = 2
	e[TICK] = 2
	c.step()
	checkRegisters(e, c, t)
}

func TestPeek(t *testing.T) {
	c := new(DCPU16)
	c.memory[0] = makeOpcode(SET, 0, PEEK) // SET A, PEEK
	e := c.Registers()
	e[A] = c.memory[0]
	e[PC] += 1
	e[TICK] = 1
	c.step()
	checkRegisters(e, c, t)
}

func TestPushPop(t *testing.T) {
	c := new(DCPU16)
	c.memory[0] = makeOpcode(SET, PUSH, 0)   // SET PUSH, A
	c.memory[1] = makeOpcode(SET, 0x01, POP) // SET B, POP
	c.register[A] = 0x7f3f
	e := c.Registers()
	e[A] = c.register[A]
	e[SP] = 0xffff
	e[PC] = 1
	e[TICK] = 1
	c.step()
	checkRegisters(e, c, t, "SET PUSH,A")
	if c.memory[e[SP]] != e[A] {
		t.Errorf("Expected value at top of stack to be %0x4d, got: %0x4d\n", e[A], c.memory[e[SP]])
	}

	e[B] = e[A]
	e[TICK] = c.tick + 1
	e[PC] = 2
	e[SP] = 0
	c.step()
	checkRegisters(e, c, t, "SET B,POP")
	if c.memory[0xffff] != e[A] {
		t.Errorf("Expected value at 0xffff to be %0x4d, got: %0x4d\n", e[A], c.memory[0xffff])
	}
}

func TestADD(t *testing.T) {
	c := new(DCPU16)
	c.memory[0] = makeOpcode(ADD, 0, 1) // ADD A,B
	e := c.Registers()

	c.pc = 0
	c.register[A] = 1
	c.register[B] = 1
	e[A] = c.register[A] + c.register[B]
	e[B] = c.register[B]
	e[TICK] = c.tick + 2
	e[PC] = 1
	c.step()
	checkRegisters(e, c, t, "ADD A,B (1,1)")

	c.pc = 0
	c.register[A] = 0xffff
	c.register[B] = 1
	e[A] = c.register[A] + c.register[B]
	e[B] = c.register[B]
	e[TICK] = c.tick + 2
	e[EX] = 1
	e[PC] = 1
	c.step()
	checkRegisters(e, c, t, "ADD A,B (0xffff,1)")

	c.pc = 0
	c.register[A] = 0x7f
	c.register[B] = 0x32
	e[A] = c.register[A] + c.register[B]
	e[B] = c.register[B]
	e[TICK] = c.tick + 2
	e[EX] = 0
	e[PC] = 1
	c.step()
	checkRegisters(e, c, t, "ADD A,B (0x7f,0x32)")
}

func TestSUB(t *testing.T) {
	c := new(DCPU16)
	c.memory[0] = makeOpcode(SUB, 0, 1) // SUB A,B
	e := c.Registers()

	c.pc = 0
	c.register[A] = 1
	c.register[B] = 1
	e[A] = c.register[A] - c.register[B]
	e[B] = c.register[B]
	e[TICK] = c.tick + 2
	e[PC] = 1
	c.step()
	checkRegisters(e, c, t, "SUB A,B (1,1)")

	c.pc = 0
	c.register[A] = 0
	c.register[B] = 1
	e[A] = c.register[A] - c.register[B]
	e[B] = c.register[B]
	e[TICK] = c.tick + 2
	e[EX] = 0xffff
	e[PC] = 1
	c.step()
	checkRegisters(e, c, t, "SUB A,B (0,1)")

	c.pc = 0
	c.register[A] = 0x7f
	c.register[B] = 0x32
	e[A] = c.register[A] - c.register[B]
	e[B] = c.register[B]
	e[TICK] = c.tick + 2
	e[EX] = 0
	e[PC] = 1
	c.step()
	checkRegisters(e, c, t, "SUB A,B (0x7f,0x32)")
}

func TestMUL(t *testing.T) {
	c := new(DCPU16)
	c.memory[0] = makeOpcode(MUL, 0, 1) // MUL A,B

	for i := 0x20; i > 0; i -= 5 {
		c.pc = 0
		c.register[A] = 0x7f3f
		c.register[B] = uint16(i)
		e := c.Registers()
		ov := uint32(c.register[A]) * uint32(c.register[B])
		e[A] = uint16(ov)
		e[EX] = uint16(ov >> 16)
		e[B] = c.register[B]
		e[PC] = 1
		e[TICK] = c.tick + 2
		c.step()
		checkRegisters(e, c, t, fmt.Sprintf("MUL A,B (0x7f3f,0x%0x4d)", i))
	}
}

func TestDIV(t *testing.T) {
	c := new(DCPU16)
	c.memory[0] = makeOpcode(DIV, 0, 1) // DIV A,B

	for i := 0x20; i > 0; i -= 5 {
		c.pc = 0
		c.register[B] = 0x7f3f
		c.register[A] = uint16(i)
		e := c.Registers()
		ov := uint32(c.register[A]) / uint32(c.register[B])
		e[B] = c.register[B]
		e[A] = uint16(ov)
		e[EX] = uint16(ov >> 16)
		e[PC] = 1
		e[TICK] = c.tick + 3
		c.step()
		checkRegisters(e, c, t, fmt.Sprintf("DIV A,B (0x7f3f,0x%0x4d)", i))
	}
}

func TestMOD(t *testing.T) {
	c := new(DCPU16)
	c.memory[0] = makeOpcode(MOD, 0, 1) // MOD A,B
	e := c.Registers()
	e[PC] = 1

	// if b == 0, a=>0
	c.pc = 0
	c.register[A] = 0xff
	c.register[B] = 0
	e[A] = 0
	e[TICK] = c.tick + 3
	c.step()
	checkRegisters(e, c, t, "MOD A,B (A=0xFF, B=0)")

	for i := 1; i < 10; i++ {
		c.pc = 0
		c.register[A] = 0xFF
		c.register[B] = uint16(i)
		e[A] = uint16(0xff % i)
		e[B] = c.register[B]
		e[TICK] = c.tick + 3
		c.step()
		checkRegisters(e, c, t, fmt.Sprintf("MOD A,B (A=0xFF, B=%d)", i))
	}

	c.pc = 0
	c.register[A] = 0x0
	c.register[B] = 0x17
	e[A] = uint16(c.register[A] % c.register[B])
	e[B] = c.register[B]
	e[TICK] = c.tick + 3
	c.step()
	checkRegisters(e, c, t, "MOD A,B (A=0, B=0x17)")
}

func TestSHL(t *testing.T) {
	c := new(DCPU16)
	c.memory[0] = makeOpcode(SHL, 0, 1) // SHR A,B
	e := c.Registers()
	e[PC] = 1
	for i := uint(0); i < 16; i++ {
		c.pc = 0
		c.register[A] = 0x0001
		c.register[B] = uint16(i)
		e[A] = uint16(0x0001 << i)
		e[B] = c.register[B]
		e[TICK] = c.tick + 1
		c.step()
		checkRegisters(e, c, t, fmt.Sprintf(" SHL A,B (A=0x0001, B=%d)", i))
	}

	// Check for proper overflow behavior
	for i := uint(0); i < 16; i++ {
		c.pc = 0
		c.register[A] = 0x8000
		c.register[B] = uint16(i)
		e[A] = c.register[A] << i
		e[B] = c.register[B]
		e[EX] = uint16((uint32(c.register[A]) << i) >> 16)
		e[TICK] = c.tick + 1
		c.step()
		checkRegisters(e, c, t, fmt.Sprintf(" SHL A,B (A=0x0001, B=%d)", i))
	}

	// make sure that overflow clears when using large values
	c.pc = 0
	c.register[A] = 0xFFFF
	c.register[B] = 0x20
	e[A] = 0
	e[B] = c.register[B]
	e[TICK] = c.tick + 1
	e[EX] = 0x0
	c.step()
	checkRegisters(e, c, t, "SHL A,B (A=0xFFFF,B=32)")
}

func TestSHR(t *testing.T) {
	c := new(DCPU16)
	c.memory[0] = makeOpcode(SHR, 0, 1) // SHR A,B
	e := c.Registers()
	e[PC] = 1
	for i := uint(0); i < 16; i++ {
		c.pc = 0
		c.register[A] = 0x8000
		c.register[B] = uint16(i)
		e[A] = uint16(0x08000 >> i)
		e[B] = c.register[B]
		e[TICK] = c.tick + 1
		c.step()
		checkRegisters(e, c, t, fmt.Sprintf(" SHR A,B (A=0x8000, B=%d)", i))
	}

	// Check for proper overflow behavior
	for i := uint(0); i < 16; i++ {
		c.pc = 0
		c.register[A] = 0x0001
		c.register[B] = uint16(i)
		e[A] = c.register[A] >> i
		e[B] = c.register[B]
		e[EX] = uint16(uint32(0x10000) >> i)
		e[TICK] = c.tick + 1
		c.step()
		checkRegisters(e, c, t, fmt.Sprintf(" SHR A,B (A=0x0001, B=%d)", i))
	}

	// make sure that overflow clears when using large values
	c.pc = 0
	c.register[A] = 0xFFFF
	c.register[B] = 0x20
	e[A] = 0
	e[B] = c.register[B]
	e[TICK] = c.tick + 1
	e[EX] = 0x0
	c.step()
	checkRegisters(e, c, t, "SHR A,B (A=0xFFFF,B=32)")
}

func TestAND(t *testing.T) {
	c := new(DCPU16)
	c.memory[0] = makeOpcode(AND, 0, 1) // AND A,B
	c.register[A] = 0x5555              // b010101010...
	c.register[B] = c.register[A]
	e := c.Registers()
	e[PC] = 1
	e[TICK] = 1
	c.step()
	checkRegisters(e, c, t, "AND A,B (A=B)")

	c.pc = 0
	c.register[A] = 0x5555
	c.register[B] = (0x5555 << 1) // b10101010...
	e[A] = 0
	e[B] = c.register[B]
	e[TICK] = c.tick + 1
	c.step()
	checkRegisters(e, c, t, "AND A,B (B = ^A)")

	c.pc = 0
	c.register[A] = 0x5050
	c.register[B] = 0x5555
	e[A] = c.register[A]
	e[B] = c.register[B]
	e[TICK] = c.tick + 1
	c.step()
	checkRegisters(e, c, t, "AND A,B (0x5050, 0x5555)")
}

func TestBOR(t *testing.T) {
	c := new(DCPU16)
	c.memory[0] = makeOpcode(BOR, 0, 1) // BOR A,B
	c.register[A] = 0x5555              // b010101010...
	c.register[B] = c.register[A]
	e := c.Registers()
	e[PC] = 1
	e[TICK] = 1
	c.step()
	checkRegisters(e, c, t, "BOR A,B (A=B)")

	c.pc = 0
	c.register[A] = 0x5555
	c.register[B] = (0x5555 << 1) // b10101010...
	e[A] = 0xffff
	e[B] = c.register[B]
	e[TICK] = c.tick + 1
	c.step()
	checkRegisters(e, c, t, "BOR A,B (B = ^A)")

	c.pc = 0
	c.register[A] = 0x0
	e[A] = c.register[B]
	e[TICK] = c.tick + 1
	c.step()
	checkRegisters(e, c, t, "BOR A,B (A = 0)")

	c.pc = 0
	c.register[A] = c.register[B]
	c.register[B] = 0
	e[A] = c.register[A]
	e[B] = c.register[B]
	e[TICK] = c.tick + 1
	c.step()
	checkRegisters(e, c, t, "BOR A,B (B = 0)")
}

func TestXOR(t *testing.T) {
	c := new(DCPU16)
	c.memory[0] = makeOpcode(XOR, 0, 1) // XOR A,B
	c.register[A] = 0x5555              // b010101010...
	c.register[B] = c.register[A]
	e := c.Registers()
	e[A] = 0
	e[PC] = 1
	e[TICK] = 1
	c.step()
	checkRegisters(e, c, t, "XOR A,B (A=B)")

	c.pc = 0
	c.register[A] = 0x5555
	c.register[B] = (0x5555 << 1) // b10101010...
	e[A] = 0xffff
	e[B] = c.register[B]
	e[TICK] = c.tick + 1
	c.step()
	checkRegisters(e, c, t, "XOR A,B (B = ^A)")

	c.pc = 0
	c.register[A] = 0x5555
	c.register[B] = 0
	e[A] = c.register[A]
	e[B] = c.register[B]
	e[TICK] = c.tick + 1
	c.step()
	checkRegisters(e, c, t, "XOR A,B (B = 0)")
}

func TestIFE(t *testing.T) {
	c := new(DCPU16)

	// check that if B==A that pc is at next instruction
	c.memory[0] = makeOpcode(IFE, 0, 1) // IFE A, B
	c.register[A] = 0x7f3f
	c.register[B] = 0x7f3f
	e := c.Registers()
	e[A] = c.register[A]
	e[B] = c.register[B]
	e[PC] = 1
	e[TICK] += 2
	c.step()
	checkRegisters(e, c, t, "IFE A==B")

	// check that if A != B that the pc is beyond next instruction, and extra cycle spent
	c.register[A] = 0x7f3f
	c.register[B] = 0
	c.pc = 0
	e[A] = c.register[A]
	e[B] = c.register[B]
	e[PC] = 2
	e[TICK] = c.tick + 3
	c.step()
	checkRegisters(e, c, t, "IFE A!=B")
}

func TestIFN(t *testing.T) {
	c := new(DCPU16)

	// check that if B != A that pc is at next instruction
	c.memory[0] = makeOpcode(IFN, 1, 0) // IFN B, A
	c.memory[1] = 0
	c.memory[2] = 0
	c.register[A] = 0
	c.register[B] = 0x7f3f
	e := c.Registers()
	e[PC] = 1
	e[TICK] = 2
	c.step()
	checkRegisters(e, c, t, "IFN A!= B")

	// check that if B == A that the pc is beyond next instruction, and extra cycle spent
	c.register[A] = 0x7f3f
	c.register[B] = 0x7f3f
	c.pc = 0
	e[A] = c.register[A]
	e[B] = c.register[B]
	e[PC] = 2
	e[TICK] = c.tick + 3
	c.step()
	checkRegisters(e, c, t, "IFN A==B")
}

func TestIFG(t *testing.T) {
	c := new(DCPU16)

	// check that if A>B that pc is at next instruction
	c.memory[0] = makeOpcode(IFG, 0, 1) // IFG A, B
	c.register[A] = 0x7f3f
	c.register[B] = c.register[A] - 1
	e := c.Registers()
	e[PC] = 1
	e[TICK] = 2
	c.step()
	checkRegisters(e, c, t, "IFG A>B")

	// check that if A=B that the pc is beyond next instruction, and extra cycle spent
	c.register[B] = c.register[A]
	c.pc = 0
	e[B] = c.register[B]
	e[PC] = 2
	e[TICK] = c.tick + 3
	c.step()
	checkRegisters(e, c, t, "IFG A=B")

	// check that if A<B that the pc is beyond next instruction, and extra cycle spent
	c.register[B] = c.register[B] + 1
	c.pc = 0
	e[B] = c.register[B]
	e[PC] = 2
	e[TICK] = c.tick + 3
	c.step()
	checkRegisters(e, c, t, "IFG A<B")
}

func TestIFB(t *testing.T) {
	c := new(DCPU16)

	// check that if A&B != 0 that pc is at next instruction
	c.memory[0] = makeOpcode(IFB, 0, 1) // IFB A, B
	c.register[A] = 0x7f3f
	c.register[B] = c.register[A]
	e := c.Registers()
	e[PC] = 1
	e[TICK] += 2
	c.step()
	checkRegisters(e, c, t, "IFB A&B != 0")

	// check that if A&B == 0 that the pc is beyond next instruction, and extra cycle spent
	c.register[B] = 0
	c.pc = 0
	e[B] = c.register[B]
	e[PC] = 2
	e[TICK] = c.tick + 3
	c.step()
	checkRegisters(e, c, t, "IFB A&B == 0")
}

func TestTickOverflow(t *testing.T) {
	c := new(DCPU16)

	c.tick = 0xfffe
	// check that if A&B != 0 that pc is at next instruction
	c.memory[0] = makeOpcode(IFB, 0, 1) // IFB A, B
	c.register[A] = 0x7f3f
	c.register[B] = c.register[A]
	e := c.Registers()
	e[PC] = 1
	e[TICK] = 0
	c.step()
	checkRegisters(e, c, t, "IFB A&B != 0")

	// check that if A&B == 0 that the pc is beyond next instruction, and extra cycle spent
	c.register[B] = 0
	c.pc = 0
	e[B] = c.register[B]
	e[PC] = 2
	e[TICK] = c.tick + 3
	c.step()
	checkRegisters(e, c, t, "IFB A&B == 0")
}

func checkRegisters(e []uint16, c *DCPU16, t *testing.T, msg ...string) {
	r := c.Registers()
	for i, v := range r {
		if v != e[i] {
			if msg == nil {
				t.Errorf("registers expected: %v, got: %v\n", e, r)
			} else {
				t.Errorf("%s: registers expected: %v, got: %v\n", msg[0], e, r)
			}
			break
		}
	}
}

func makeOpcode(o, b, a int) uint16 {
	if o < 0 || o > 0x1f {
		panic("Invalid opcode found in test case")
	}
	if a < 0 || a > 0x3f {
		panic("Invalid a address mode found in test case")
	}
	if b < 0 || b > 0x1f {
		panic("Invalid b address mode found in test case")
	}
	return uint16((a<<ARGA_SHIFT)&ARGA_MASK | (b<<ARGB_SHIFT)&ARGB_MASK | (o & OPCODE_MASK))
}
