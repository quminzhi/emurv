package sim

import "fmt"

type UART struct{}

func NewUART() *UART { return &UART{} }

// Tx prints the byte (teaching: simplest “serial console”)
func (u *UART) Tx(b uint8) {
	fmt.Printf("%c", b)
}
