package dcpu16

import "testing"

func TestWriteAndRead(t *testing.T) {
	Write(0, []uint16{0x7c01, 0x0030, 0x7de1})
	if memory[0] != 0x7c01 && memory[1] != 0x0030 && memory[2] != 0x7de1 {
		t.Errorf("Expected memory to equal written values, got: %v\n", memory[0:4])
	}
	m := Read(0, 4)
	if m[0] != 0x7c01 && m[1] != 0x0030 && m[2] != 0x7de1 {
		t.Errorf("Expected memory to equal written values, got: %v\n", m)
	}
}

func TestRegisters(t *testing.T) {
	Reset()
	// expect the registers to be zeroed
	e := make([]uint16, regSize)
	checkRegisters(e, t)
}

func TestSetA(t *testing.T) {
	Reset()
	Write(0, []uint16{0x7c01, 0x0030}) // SET A, 0x30
	e := Registers()
	e[PC] += 2
	e[TICK] += 3
	e[A] = 0x30
	step()
	checkRegisters(e, t)
}

func TestSetAllRegisters(t *testing.T) {
	for i := 0; i <= 7; i++ {
		Reset()
		memory[0] = makeOpcode(0x01, i, 0x1f) // SET I, 0x0030
		memory[1] = 0x0030
		e := Registers()
		e[PC] += 2
		e[TICK] += 3
		e[i] = 0x30
		step()
		checkRegisters(e, t)
	}
}

func TestSetPC(t *testing.T) {
	Reset()
	Write(0, []uint16{0x7dc1, 0x0030}) // SET PC, 0x30
	e := Registers()
	e[PC] = 0x30
	e[TICK] += 3
	step()
	checkRegisters(e, t)
}

func TestSetO(t *testing.T) {
	Reset()
	Write(0, []uint16{0x7dd1, 0x0030}) // SET O, 0x30
	e := Registers()
	e[O] = 0x30
	e[PC] += 2
	e[TICK] += 3
	step()
	checkRegisters(e, t)
}

func TestSetSP(t *testing.T) {
	Reset()
	Write(0, []uint16{0x7db1, 0x0030}) // SET SP, 0x30
	e := Registers()
	e[SP] = 0x30
	e[PC] += 2
	e[TICK] += 3
	step()
	checkRegisters(e, t)
}

func TestSetRegisterIndirect(t *testing.T) {
	Reset()
	Write(0, []uint16{0x2811, 0xabca}) // SET B, [C]
	register[C] = 1
	e := Registers()
	e[B] = memory[1]
	e[C] = 1
	e[PC] += 1
	e[TICK] += 1
	step()
	checkRegisters(e, t)
}

func TestSetRegisterIndirectOffset(t *testing.T) {
	Reset()
	Write(0, []uint16{0x4011, 0x0000}) // SET B, [0x0000+A]
	e := Registers()
	e[B] = memory[0]
	e[PC] += 2
	e[TICK] += 3
	step()
	checkRegisters(e, t)
}

func TestSetIndirect(t *testing.T) {
	Reset()
	Write(0, []uint16{0x7811, 0x0002, 0x7ce3}) // SET B, [0x0002]
	e := Registers()
	e[B] = memory[2]
	e[PC] += 2
	e[TICK] += 3
	step()
	checkRegisters(e, t)
}

func TestSetAllShortLiterals(t *testing.T) {
	for i := 0; i <= 0x1f; i++ {
		Reset()
		memory[0] = makeOpcode(0x01, 0, 0x20+i) // SET A, i
		e := Registers()
		e[PC] += 1
		e[TICK] += 1
		e[A] = uint16(i)
		step()
		checkRegisters(e, t)
	}
}

func TestSetAssignLiteral(t *testing.T) {
	Reset()
	Write(0, []uint16{0x7ff1, 0x0030}) // SET 0x1f, 0x30
	e := Registers()
	e[PC] += 2
	e[TICK] += 3
	step()
	checkRegisters(e, t)
}

func TestPeek(t *testing.T) {
	Reset()
	Write(0, []uint16{0x6401}) // SET A, PEEK
	e := Registers()
	e[A] = memory[0]
	e[PC] += 1
	e[TICK] += 1
	step()
	checkRegisters(e, t)
}

func TestPushPop(t *testing.T) {
	Reset()
	Write(0, []uint16{0x01a1, 0x6011}) // SET PUSH, A; SET B, POP
	register[A] = 0x7f3f
	e := Registers()
	e[A] = 0x7f3f
	e[SP] = 0xffff
	e[PC] += 1
	e[TICK] += 1
	step()
	checkRegisters(e, t)
	if memory[0xffff] != 0x7f3f {
		t.Errorf("Expected stack to be 0x7f3f, got: %v\n", memory[0xffff])
	}
	e[B] = e[A]
	e[TICK] += 1
	e[PC] += 1
	e[SP] = 0
	step()
	checkRegisters(e, t)
	if memory[0xffff] != 0x7f3f {
		t.Errorf("Expected stack to be 0x7f3f, got: %v\n", memory[0xffff])
	}
}

func checkRegisters(e []uint16, t *testing.T) {
	r := Registers()
	for i, v := range r {
		if v != e[i] {
			t.Errorf("Expected registers to be %v, got %v\n", e, r)
		}
	}
}

func makeOpcode(o, a, b int) uint16 {
	return uint16((b&0x3f)<<10 | (a&0x3f)<<4 | o&0x0f)
}
