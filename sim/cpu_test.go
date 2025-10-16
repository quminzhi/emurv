package sim

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"strings"
	"testing"
)

/* ----------------- helpers to encode RV32I instructions ----------------- */

// R-type
func encR(op, rd, f3, rs1, rs2, f7 uint32) uint32 {
	return (f7 << 25) | (rs2 << 20) | (rs1 << 15) | (f3 << 12) | (rd << 7) | op
}

// I-type (imm is 12-bit signed)
func encI(op, rd, f3, rs1 uint32, imm int32) uint32 {
	u := uint32(imm) & 0xFFF
	return (u << 20) | (rs1 << 15) | (f3 << 12) | (rd << 7) | op
}

// S-type (imm is 12-bit signed)
func encS(op, f3, rs1, rs2 uint32, imm int32) uint32 {
	u := uint32(imm) & 0xFFF
	immhi := (u >> 5) & 0x7F
	immlo := u & 0x1F
	return (immhi << 25) | (rs2 << 20) | (rs1 << 15) | (f3 << 12) | (immlo << 7) | op
}

// B-type (imm is 13-bit signed, multiples of 2)
func encB(op, f3, rs1, rs2 uint32, imm int32) uint32 {
	u := uint32(imm)
	b12 := (u >> 12) & 0x1
	b10_5 := (u >> 5) & 0x3F
	b4_1 := (u >> 1) & 0xF
	b11 := (u >> 11) & 0x1
	return (b12 << 31) | (b10_5 << 25) | (rs2 << 20) | (rs1 << 15) |
		(f3 << 12) | (b4_1 << 8) | (b11 << 7) | op
}

// U-type (imm20 is the upper 20 bits)
func encU(op, rd, imm20 uint32) uint32 {
	return (imm20 << 12) | (rd << 7) | op
}

func writeWords(t *testing.T, ram *RAM, base uint32, words ...uint32) {
	t.Helper()
	buf := new(bytes.Buffer)
	for _, w := range words {
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], w)
		buf.Write(b[:])
	}
	if err := ram.WriteBytes(base, buf.Bytes()); err != nil {
		t.Fatalf("WriteWords: %v", err)
	}
}

func runUntilHalt(cpu *CPU, max int) {
	for i := 0; i < max; i++ {
		if !cpu.Step() {
			return
		}
	}
}

/* ------------------------------ tests ------------------------------ */

func TestUARTAndEcallHalt(t *testing.T) {
	ram := NewRAM(1 * 1024 * 1024) // 1 MiB
	uart := NewUART()
	bus := NewBus(ram, uart)
	cpu := NewCPU(bus)

	// Program:
	//   LUI  x1, 0x10000        ; x1 = 0x10000000 (UART base)
	//   ADDI x2, x0, 'A'
	//   SB   x2, 0(x1)          ; write 'A' to UART TX
	//   ECALL                   ; halt in our emulator
	instLUIx1 := encU(0x37, 1, 0x10000)           // 0x100000b7
	instADDI := encI(0x13, 2, 0x0, 0, int32('A')) // addi x2,x0,'A'
	instSB := encS(0x23, 0x0, 1, 2, 0)            // sb x2,0(x1)
	instECALL := uint32(0x00000073)

	writeWords(t, ram, 0,
		instLUIx1,
		instADDI,
		instSB,
		instECALL,
	)

	// Capture stdout (UART prints to stdout; ECALL prints "[halt] ECALL")
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	runUntilHalt(cpu, 10)

	w.Close()
	os.Stdout = oldStdout
	out, _ := io.ReadAll(r)

	s := string(out)
	if !strings.Contains(s, "A") {
		t.Fatalf("expected UART output to contain 'A', got: %q", s)
	}
	if !strings.Contains(s, "[halt] ECALL") {
		t.Fatalf("expected halt message, got: %q", s)
	}
}

func TestLBSignExtension(t *testing.T) {
	ram := NewRAM(1 * 1024 * 1024)
	uart := NewUART()
	bus := NewBus(ram, uart)
	cpu := NewCPU(bus)

	// Place 0xFF at address 0x100
	if !ram.Write8(0x100, 0xFF) {
		t.Fatal("failed to write test byte")
	}

	// Program:
	//   ADDI x3, x0, 0x100      ; base = 0x100
	//   LB   x4, 0(x3)          ; should sign-extend 0xFF -> 0xFFFFFFFF
	//   ECALL
	instAddiBase := encI(0x13, 3, 0x0, 0, 0x100) // addi x3,x0,0x100
	instLB := encI(0x03, 4, 0x0, 3, 0)           // lb x4,0(x3)
	instECALL := uint32(0x00000073)

	writeWords(t, ram, 0, instAddiBase, instLB, instECALL)
	runUntilHalt(cpu, 10)

	if cpu.Reg[4] != 0xFFFFFFFF {
		t.Fatalf("LB sign-ext failed: got 0x%08x, want 0xFFFFFFFF", cpu.Reg[4])
	}
}

func TestBEQBranchSkips(t *testing.T) {
	ram := NewRAM(1 * 1024 * 1024)
	uart := NewUART()
	bus := NewBus(ram, uart)
	cpu := NewCPU(bus)

	// Program:
	//   ADDI x5, x0, 1
	//   BEQ  x5, x5, +8         ; skip next instruction (8 bytes)
	//   ADDI x6, x0, 99         ; should be skipped
	//   ADDI x6, x0, 7          ; should execute
	//   ECALL
	instADDIx5 := encI(0x13, 5, 0x0, 0, 1)
	instBEQskip := encB(0x63, 0x0, 5, 5, 8)
	instADDIx6a := encI(0x13, 6, 0x0, 0, 99)
	instADDIx6b := encI(0x13, 6, 0x0, 0, 7)
	instECALL := uint32(0x00000073)

	writeWords(t, ram, 0, instADDIx5, instBEQskip, instADDIx6a, instADDIx6b, instECALL)
	runUntilHalt(cpu, 20)

	if cpu.Reg[6] != 7 {
		t.Fatalf("branch failed: x6=0x%x, want 7", cpu.Reg[6])
	}
}
