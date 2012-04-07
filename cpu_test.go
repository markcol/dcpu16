package dcpu16

import "testing"

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
	c.Write(0, []uint16{0x7c01, 0x0030}) // SET A, 0x30
	e := c.Registers()
	e[PC] += 2
	e[TICK] += 3
	e[A] = 0x30
	c.step()
	checkRegisters(e, c, t)
}

func TestSetAllRegisters(t *testing.T) {
	c := new(DCPU16)
	for i := 0; i <= 7; i++ {
		c.memory[0] = makeOpcode(0x01, i, 0x1f) // SET I, 0x0030
		c.memory[1] = 0x0030
		c.pc = 0
		e := c.Registers()
		e[PC] = 2
		e[TICK] += 3
		e[i] = 0x30
		c.step()
		checkRegisters(e, c, t)
	}
}

func TestSetPC(t *testing.T) {
	c := new(DCPU16)
	c.Write(0, []uint16{0x7dc1, 0x0030}) // SET PC, 0x30
	e := c.Registers()
	e[PC] = 0x30
	e[TICK] += 3
	c.step()
	checkRegisters(e, c, t)
}

func TestSetO(t *testing.T) {
	c := new(DCPU16)
	c.Write(0, []uint16{0x7dd1, 0x0030}) // SET O, 0x30
	e := c.Registers()
	e[O] = 0x30
	e[PC] += 2
	e[TICK] += 3
	c.step()
	checkRegisters(e, c, t)
}

func TestSetSP(t *testing.T) {
	c := new(DCPU16)
	c.Write(0, []uint16{0x7db1, 0x0030}) // SET SP, 0x30
	e := c.Registers()
	e[SP] = 0x30
	e[PC] += 2
	e[TICK] += 3
	c.step()
	checkRegisters(e, c, t)
}

func TestSetRegisterIndirect(t *testing.T) {
	c := new(DCPU16)
	c.Write(0, []uint16{0x2811, 0xabca}) // SET B, [C]
	c.register[C] = 1
	e := c.Registers()
	e[B] = c.memory[1]
	e[C] = 1
	e[PC] += 1
	e[TICK] += 1
	c.step()
	checkRegisters(e, c, t)
}

func TestSetRegisterIndirectOffset(t *testing.T) {
	c := new(DCPU16)
	c.Write(0, []uint16{0x4011, 0x0000}) // SET B, [0x0000+A]
	e := c.Registers()
	e[B] = c.memory[0]
	e[PC] += 2
	e[TICK] += 3
	c.step()
	checkRegisters(e, c, t)
}

func TestSetIndirect(t *testing.T) {
	c := new(DCPU16)
	c.Write(0, []uint16{0x7811, 0x0002, 0x7ce3}) // SET B, [0x0002]
	e := c.Registers()
	e[B] = c.memory[2]
	e[PC] += 2
	e[TICK] += 3
	c.step()
	checkRegisters(e, c, t)
}

func TestSetAllShortLiterals(t *testing.T) {
	c := new(DCPU16)
	for i := 0; i <= 0x1f; i++ {
		c.pc = 0
		c.memory[0] = makeOpcode(0x01, 0, 0x20+i) // SET A, i
		e := c.Registers()
		e[PC] = 1
		e[TICK] += 1
		e[A] = uint16(i)
		c.step()
		checkRegisters(e, c, t)
	}
}

func TestSetAssignLiteral(t *testing.T) {
	c := new(DCPU16)
	c.Write(0, []uint16{0x7ff1, 0x0030}) // SET 0x1f, 0x30
	e := c.Registers()
	e[PC] += 2
	e[TICK] += 3
	c.step()
	checkRegisters(e, c, t)
}

func TestPeek(t *testing.T) {
	c := new(DCPU16)
	c.Write(0, []uint16{0x6401}) // SET A, PEEK
	e := c.Registers()
	e[A] = c.memory[0]
	e[PC] += 1
	e[TICK] += 1
	c.step()
	checkRegisters(e, c, t)
}

func TestPushPop(t *testing.T) {
	c := new(DCPU16)
	c.Write(0, []uint16{0x01a1, 0x6011}) // SET PUSH, A; SET B, POP
	c.register[A] = 0x7f3f
	e := c.Registers()
	e[A] = 0x7f3f
	e[SP] = 0xffff
	e[PC] += 1
	e[TICK] += 1
	c.step()
	checkRegisters(e, c, t)
	if c.memory[0xffff] != 0x7f3f {
		t.Errorf("Expected stack to be 0x7f3f, got: %v\n", c.memory[0xffff])
	}
	e[B] = e[A]
	e[TICK] += 1
	e[PC] += 1
	e[SP] = 0
	c.step()
	checkRegisters(e, c, t)
	if c.memory[0xffff] != 0x7f3f {
		t.Errorf("Expected stack to be 0x7f3f, got: %v\n", c.memory[0xffff])
	}
}

func TestIFG(t *testing.T) {
	c := new(DCPU16)

	// check that if A&B != 0 that pc is at next instruction
	c.memory[0] = makeOpcode(0xf, 0, 1) // IFB A, B
	c.register[A] = 0x7f3f
	c.register[B] = c.register[A]
	e := c.Registers()
	e[A] = 0x7f3f
	e[B] = e[A]
	e[PC] = 1
	e[TICK] += 2
	c.step()
	checkRegisters(e, c, t)

	// check that if A&B == 0 that the pc is beyond next instruction, and extra cycle spent
	c.register[B] = 0
	c.pc = 0
	e[B] = 0
	e[PC] = 2
	e[TICK] += 3
	c.step()
	checkRegisters(e, c, t)
}

func checkRegisters(e []uint16, c *DCPU16, t *testing.T) {
	r := c.Registers()
	for i, v := range r {
		if v != e[i] {
			t.Errorf("Expected registers to be %v, got %v\n", e, r)
		}
	}
}

func makeOpcode(o, a, b int) uint16 {
	return uint16((b&0x3f)<<10 | (a&0x3f)<<4 | o&0x0f)
}
