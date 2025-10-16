package sim

// Simple address map and read/write helpers.
// RAM:       0x0000_0000 .. size-1
// UART:      0x1000_0000 .. 0x1000_00FF (TX at +0x00, STATUS at +0x04)

const (
	UARTBase   = 0x10000000
	UARTSize   = 0x100
	UARTTx     = UARTBase + 0x00
	UARTStatus = UARTBase + 0x04
)

type Bus struct {
	ram  *RAM
	uart *UART
}

func NewBus(ram *RAM, uart *UART) *Bus {
	return &Bus{ram: ram, uart: uart}
}

func (b *Bus) Read8(addr uint32) (uint8, bool) {
	// UART MMIO
	if addr >= UARTBase && addr < UARTBase+UARTSize {
		switch addr {
		case UARTStatus:
			return 1, true // always ready
		default:
			return 0, true
		}
	}

	// RAM
	return b.ram.Read8(addr)
}

func (b *Bus) Write8(addr uint32, v uint8) bool {
	if addr >= UARTBase && addr < UARTBase+UARTSize {
		switch addr {
		case UARTTx:
			b.uart.Tx(v)
			return true
		default:
			return true
		}
	}
	return b.ram.Write8(addr, v)
}

func (b *Bus) Read32(addr uint32) (uint32, bool) {
	// Compose 4 bytes via Read8 (handles MMIO too)
	b0, ok := b.Read8(addr)
	if !ok {
		return 0, false
	}
	b1, ok := b.Read8(addr + 1)
	if !ok {
		return 0, false
	}
	b2, ok := b.Read8(addr + 2)
	if !ok {
		return 0, false
	}
	b3, ok := b.Read8(addr + 3)
	if !ok {
		return 0, false
	}
	return uint32(b0) | uint32(b1)<<8 | uint32(b2)<<16 | uint32(b3)<<24, true
}

func (b *Bus) Write32(addr uint32, v uint32) bool {
	// Special-case UART TX convenience: word store prints low byte
	if addr == UARTTx {
		b.uart.Tx(uint8(v & 0xFF))
		return true
	}
	return b.Write8(addr, uint8(v&0xFF)) &&
		b.Write8(addr+1, uint8((v>>8)&0xFF)) &&
		b.Write8(addr+2, uint8((v>>16)&0xFF)) &&
		b.Write8(addr+3, uint8((v>>24)&0xFF))
}
