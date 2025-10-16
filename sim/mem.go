package sim

import (
	"fmt"
	"os"
)

type RAM struct {
	data []byte
}

func NewRAM(size uint64) *RAM {
	return &RAM{data: make([]byte, size)}
}

func (m *RAM) Size() uint32 {
	return uint32(len(m.data))
}

func (m *RAM) Read8(addr uint32) (uint8, bool) {
	if addr >= uint32(len(m.data)) {
		return 0, false
	}
	return m.data[addr], true
}

func (m *RAM) Write8(addr uint32, v uint8) bool {
	if addr >= uint32(len(m.data)) {
		return false
	}
	m.data[addr] = v
	return true
}

func (m *RAM) LoadFlat(path string, base uint32) error {
	f, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if base+uint32(len(f)) > uint32(len(m.data)) {
		return fmt.Errorf("flat image too large for RAM")
	}
	copy(m.data[base:], f)
	return nil
}

func (m *RAM) WriteBytes(addr uint32, buf []byte) error {
	if addr+uint32(len(buf)) > uint32(len(m.data)) {
		return fmt.Errorf("write beyond RAM")
	}
	copy(m.data[addr:], buf)
	return nil
}
