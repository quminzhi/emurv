package main

import (
	"flag"
	"fmt"
	"os"

	"emurv/sim"
)

func main() {
	elfPath := flag.String("elf", "", "ELF file to load")
	binPath := flag.String("bin", "", "Flat binary to load at 0x0")
	steps := flag.Int("steps", 10_000_000, "Max steps")
	trace := flag.Bool("trace", false, "Print each instruction (teaching mode)")
	memMiB := flag.Int("mem", 16, "RAM MiB (default 16)")
	startPC := flag.Uint("pc", 0, "Override start PC (0 keeps loader entry/reset)")

	flag.Parse()

	// Build the machine
	ram := sim.NewRAM(uint64(*memMiB) * 1024 * 1024)
	uart := sim.NewUART()
	bus := sim.NewBus(ram, uart)
	cpu := sim.NewCPU(bus)
	cpu.Trace = *trace

	// Load program
	switch {
	case *elfPath != "":
		entry, err := sim.LoadELF(*elfPath, ram)
		if err != nil {
			fmt.Fprintln(os.Stderr, "ELF load error:", err)
			os.Exit(1)
		}
		cpu.PC = uint32(entry)
	case *binPath != "":
		if err := ram.LoadFlat(*binPath, 0); err != nil {
			fmt.Fprintln(os.Stderr, "BIN load error:", err)
			os.Exit(1)
		}
		cpu.PC = 0
	default:
		fmt.Fprintln(os.Stderr, "No program provided. Use -elf or -bin.")
		os.Exit(2)
	}

	if *startPC != 0 {
		cpu.PC = uint32(*startPC)
	}

	// Run
	for i := 0; i < *steps; i++ {
		if !cpu.Step() {
			break
		}
	}
}
