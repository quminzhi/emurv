package sim

import "fmt"

// Minimal RV32I subset, with LB/LBU/SB added for UART-bytes demos.
// Treat any ECALL as "halt".

type CPU struct {
	Reg   [32]uint32
	PC    uint32
	Bus   *Bus
	Trace bool
}

func NewCPU(bus *Bus) *CPU { return &CPU{Bus: bus} }

func (c *CPU) readReg(i uint32) uint32 {
	if i == 0 {
		return 0
	}
	return c.Reg[i]
}

func (c *CPU) writeReg(i uint32, v uint32) {
	if i != 0 {
		c.Reg[i] = v
	}
}

func (c *CPU) fetch() (uint32, bool) {
	return c.Bus.Read32(c.PC)
}

func (c *CPU) Step() bool {
	inst, ok := c.fetch()
	if !ok {
		fmt.Println("\n[trap] fetch OOB")
		return false
	}
	op := inst & 0x7F
	rd := (inst >> 7) & 0x1F
	f3 := (inst >> 12) & 0x7
	rs1 := (inst >> 15) & 0x1F
	rs2 := (inst >> 20) & 0x1F
	f7 := (inst >> 25) & 0x7F

	nextPC := c.PC + 4

	if c.Trace {
		fmt.Printf("pc=%08x inst=%08x\n", c.PC, inst)
	}

	switch op {
	case 0x37: // LUI
		c.writeReg(rd, uint32(immU(inst)))
	case 0x17: // AUIPC
		c.writeReg(rd, uint32(int32(c.PC)+immU(inst)))
	case 0x6F: // JAL
		imm := uint32(immJ(inst))
		c.writeReg(rd, c.PC+4)
		nextPC = c.PC + imm
	case 0x67: // JALR
		imm := uint32(immI(inst))
		tgt := (c.readReg(rs1) + imm) &^ 1
		c.writeReg(rd, c.PC+4)
		nextPC = tgt

	case 0x63: // BRANCH
		a := c.readReg(rs1)
		b := c.readReg(rs2)
		imm := uint32(immB(inst))
		switch f3 {
		case 0x0: // BEQ
			if a == b {
				nextPC = c.PC + imm
			}
		case 0x1: // BNE
			if a != b {
				nextPC = c.PC + imm
			}
		case 0x4: // BLT
			if int32(a) < int32(b) {
				nextPC = c.PC + imm
			}
		case 0x5: // BGE
			if int32(a) >= int32(b) {
				nextPC = c.PC + imm
			}
		case 0x6: // BLTU
			if a < b {
				nextPC = c.PC + imm
			}
		case 0x7: // BGEU
			if a >= b {
				nextPC = c.PC + imm
			}
		default:
			fmt.Printf("[warn] BRANCH f3=%d\n", f3)
		}

	case 0x03: // LOAD
		base := c.readReg(rs1)
		addr := base + uint32(immI(inst))
		switch f3 {
		case 0x0: // LB
			b, ok := c.Bus.Read8(addr)
			if !ok {
				fmt.Println("\n[trap] LB OOB")
				return false
			}
			c.writeReg(rd, uint32(int32(int8(b))))
		case 0x4: // LBU
			b, ok := c.Bus.Read8(addr)
			if !ok {
				fmt.Println("\n[trap] LBU OOB")
				return false
			}
			c.writeReg(rd, uint32(b))
		case 0x2: // LW
			w, ok := c.Bus.Read32(addr)
			if !ok {
				fmt.Println("\n[trap] LW OOB")
				return false
			}
			c.writeReg(rd, w)
		default:
			fmt.Printf("[warn] LOAD f3=%d\n", f3)
		}

	case 0x23: // STORE
		base := c.readReg(rs1)
		addr := base + uint32(immS(inst))
		switch f3 {
		case 0x0: // SB
			v := uint8(c.readReg(rs2) & 0xFF)
			if !c.Bus.Write8(addr, v) {
				fmt.Println("\n[trap] SB OOB")
				return false
			}
		case 0x2: // SW
			v := c.readReg(rs2)
			if !c.Bus.Write32(addr, v) {
				fmt.Println("\n[trap] SW OOB")
				return false
			}
		default:
			fmt.Printf("[warn] STORE f3=%d\n", f3)
		}

	case 0x13: // OP-IMM
		a := c.readReg(rs1)
		imm := uint32(immI(inst))
		switch f3 {
		case 0x0: // ADDI
			c.writeReg(rd, a+uint32(int32(imm)))
		case 0x4: // XORI
			c.writeReg(rd, a^imm)
		case 0x6: // ORI
			c.writeReg(rd, a|imm)
		case 0x7: // ANDI
			c.writeReg(rd, a&imm)
		case 0x1: // SLLI
			sh := (imm & 0x1F)
			c.writeReg(rd, a<<sh)
		case 0x5:
			if (imm>>10)&0x3F == 0x00 { // SRLI
				c.writeReg(rd, a>>(imm&0x1F))
			} else if (imm>>10)&0x3F == 0x10 { // SRAI
				c.writeReg(rd, uint32(int32(a)>>(imm&0x1F)))
			} else {
				fmt.Printf("[warn] OP-IMM funct5?\n")
			}
		default:
			fmt.Printf("[warn] OP-IMM f3=%d\n", f3)
		}

	case 0x33: // OP
		a := c.readReg(rs1)
		b := c.readReg(rs2)
		switch f3 {
		case 0x0:
			if f7 == 0x20 { // SUB
				c.writeReg(rd, a-b)
			} else { // ADD
				c.writeReg(rd, a+b)
			}
		case 0x4: // XOR
			c.writeReg(rd, a^b)
		case 0x6: // OR
			c.writeReg(rd, a|b)
		case 0x7: // AND
			c.writeReg(rd, a&b)
		case 0x1: // SLL
			c.writeReg(rd, a<<(b&0x1F))
		case 0x5: // SRL/SRA
			if f7 == 0x20 {
				c.writeReg(rd, uint32(int32(a)>>(b&0x1F)))
			} else {
				c.writeReg(rd, a>>(b&0x1F))
			}
		case 0x2: // SLT
			if int32(a) < int32(b) {
				c.writeReg(rd, 1)
			} else {
				c.writeReg(rd, 0)
			}
		case 0x3: // SLTU
			if a < b {
				c.writeReg(rd, 1)
			} else {
				c.writeReg(rd, 0)
			}
		default:
			fmt.Printf("[warn] OP f3=%d f7=0x%x\n", f3, f7)
		}

	case 0x73: // SYSTEM
		// We treat ECALL as a clean halt for teaching.
		fmt.Println("\n[halt] ECALL")
		return false

	default:
		fmt.Printf("\n[warn] unsupported opcode 0x%x at pc=%08x\n", op, c.PC)
	}

	c.PC = nextPC
	c.Reg[0] = 0
	return true
}
