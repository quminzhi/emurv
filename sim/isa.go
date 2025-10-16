package sim

func signExtend(v uint32, bits uint) int32 {
	shift := 32 - bits
	return int32(v<<shift) >> shift
}

func immI(inst uint32) int32 { return signExtend(inst>>20, 12) }

func immS(inst uint32) int32 {
	low := (inst >> 7) & 0x1F
	hi := (inst >> 25) & 0x7F
	return signExtend((hi<<5)|low, 12)
}

func immB(inst uint32) int32 {
	// [12|10:5|4:1|11] << 1
	imm := ((inst>>31)&1)<<12 |
		((inst>>25)&0x3F)<<5 |
		((inst>>8)&0xF)<<1 |
		((inst>>7)&1)<<11
	return signExtend(uint32(imm), 13)
}

func immU(inst uint32) int32 { return int32(inst & 0xFFFFF000) }

func immJ(inst uint32) int32 {
	// [20|10:1|11|19:12] << 1
	imm := ((inst>>31)&1)<<20 |
		((inst>>21)&0x3FF)<<1 |
		((inst>>20)&1)<<11 |
		((inst>>12)&0xFF)<<12
	return signExtend(uint32(imm), 21)
}
