package sim

import (
	"debug/elf"
	"fmt"
)

// LoadELF maps PT_LOAD segments into RAM at their paddr/vaddr (we assume
// an identity physical mapping for teaching). Returns entry address.
func LoadELF(path string, ram *RAM) (entry uint64, err error) {
	f, err := elf.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	for _, ph := range f.Progs {
		if ph.Type != elf.PT_LOAD {
			continue
		}
		// Read the segment bytes
		buf := make([]byte, ph.Memsz)
		if ph.Filesz > 0 {
			if _, err := ph.ReadAt(buf[:ph.Filesz], 0); err != nil {
				return 0, fmt.Errorf("read segment: %w", err)
			}
		}
		// Zero-fill remainder (already zeroed by make)
		addr := uint32(ph.Vaddr) // RV32I
		if err := ram.WriteBytes(addr, buf); err != nil {
			return 0, fmt.Errorf("map segment @0x%x: %w", addr, err)
		}
	}

	return uint64(f.Entry), nil
}
