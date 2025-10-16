# emurv — a tiny RV32I emulator in Go (for teaching)

**emurv** is a deliberately small, readable emulator for the **RISC-V RV32I** base ISA.  
It models:

- a 32-bit CPU (registers + fetch/decode/execute loop)  
- RAM  
- a memory-mapped **UART** (so programs can “print”)  
- an **ELF loader** (so you can compile with a standard RISC-V toolchain)

It’s built to be easy to extend in class/labs.

---

## Features (what’s implemented)

- **CPU state:** 32 regs (`x0..x31`), program counter
- **ISA subset:**  
  - Control flow: `LUI, AUIPC, JAL, JALR, BEQ, BNE, BLT, BGE, BLTU, BGEU`  
  - Loads/stores: `LB, LBU, LW, SB, SW`  
  - ALU (imm): `ADDI, XORI, ORI, ANDI, SLLI, SRLI, SRAI`  
  - ALU (reg): `ADD, SUB, XOR, OR, AND, SLL, SRL, SRA, SLT, SLTU`  
  - System: `ECALL` (treated as **halt**)
- **Memory map:**  
  - RAM at `0x00000000 … (size-1)`  
  - **UART** at `0x1000_0000` (TX: `+0x00`, STATUS “ready”: `+0x04`)
- **I/O:** writing a byte/word to `UART_TX` prints that byte to stdout
- **ELF loader:** maps `PT_LOAD` segments; starts at ELF entry
- **Unit tests:** tiny, self-contained instruction/IO tests (`go test`)

---

## Project layout

```
emurv/
├─ go.mod
├─ cmd/
│  └─ emurv/
│     └─ main.go            # CLI: loads program, runs CPU
└─ sim/
   ├─ cpu.go                # CPU core (decode + execute)
   ├─ isa.go                # immediate decode helpers
   ├─ bus.go                # address map + MMIO dispatch
   ├─ mem.go                # RAM
   ├─ uart.go               # UART “device”
   └─ elf.go                # ELF loader
examples/
└─ uart_hello/
   ├─ hello.s               # prints "Hello, RISC-V!\n"
   └─ linker.ld             # simple RV32 link script (at 0x0)
Makefile
```

---

## Prerequisites

- **Go 1.22+**
- A **RISC-V GNU toolchain** (for building the example)
  - macOS (Homebrew):
    ```bash
    brew tap riscv-software-src/riscv
    brew install riscv-gnu-toolchain
    ```
  - If your toolchain prefix is `riscv32-unknown-elf`, set `RISCV_PREFIX=riscv32-unknown-elf` when using `make`.

---

## Quick start

1) **Build the example program**
```bash
make
# generates:
# examples/uart_hello/uart_hello.elf
# examples/uart_hello/uart_hello.bin
```

2) **Run the emulator (ELF is recommended)**
```bash
go run ./cmd/emurv -elf examples/uart_hello/uart_hello.elf -trace
```

Expected output:
```
pc=00000000 inst=...   # (trace lines)
Hello, RISC-V!
[halt] ECALL
```

3) **Run unit tests**
```bash
go test ./...
```

---

## CLI usage

```bash
go run ./cmd/emurv   [-elf path/to/program.elf]   [-bin path/to/flat.bin]   [-pc 0xXXXXXXXX]   [-mem 16]   [-steps 10000000]   [-trace]
```

- `-elf` **or** `-bin` is required (ELF sets start PC to the ELF entry; BIN loads at 0x0 with `PC=0`).
- `-trace` prints each instruction and PC (great for demos).
- `-mem` sets RAM size in MiB (default 16).

---

## How it works (quick mental model)

```
User program (ELF/BIN)
        │
     [Loader]
        │
+---------------------+        +--------------------+
|      CPU            |        |        Bus         |
|  regs[] + PC        |<------>|  RAM + MMIO map    |
|  fetch/decode/exec  |        |  ├─ RAM            |
+---------------------+        |  └─ UART (@0x1000_0000)
                               +--------------------+

- CPU fetches a 32-bit inst from PC via Bus.
- CPU decodes opcode/funct fields and executes.
- Loads/stores go through Bus → RAM or UART.
- Writing to UART_TX prints a byte to stdout.
- ECALL prints “[halt] ECALL” and stops.
```

---

## Teaching labs / exercises (suggested order)

1. **Trace reading**  
   Run with `-trace`. Identify `LUI/ADDI/SB` creating a character print.

2. **Add an ALU instruction**  
   Implement `ORI`/`XORI` (if any missing) by filling a case in `cpu.go` under `OP-IMM`/`OP`.

3. **Add load/store variants**  
   Implement `LH`, `LHU`, `SH`.  
   – Update decode in `LOAD/STORE` cases and add tests.

4. **Add a second device**  
   Create `timer.go` with MMIO at `0x2000_0000`, incrementing a counter; expose a “ready” flag and read value.

5. **ELF exploration**  
   Print ELF segments on load (`elf.go`) to show addresses and sizes; explain `PT_LOAD`.

6. **Exceptions (bonus)**  
   Detect out-of-bounds accesses and collect a simple “trap” struct (pc, cause, addr).

7. **Interrupts (advanced)**  
   Simulate a timer interrupt: on “ready,” branch into an ISR (you’ll sketch a tiny CSR model).

Each exercise should come with a tiny unit test similar to `cpu_test.go`.

---

## How to add an instruction (pattern)

1. Find the switch in `sim/cpu.go` on the **opcode** (`op := inst & 0x7F`).  
2. Inside the matching case, use `f3`, `f7`, `rs1`, `rs2`, `rd`, and immediates from `isa.go` (e.g., `immI(inst)`).  
3. Read operands with `c.readReg(rs1)` / `c.readReg(rs2)`.  
4. Compute the result and write back with `c.writeReg(rd, value)`.  
   (Remember: **x0 stays 0**; the helper already enforces this.)  
5. Add a minimal unit test that synthesizes the instruction encoding.

---

## How to add a device (MMIO)

1. Pick a base address range (e.g., `0x2000_0000 .. +0xFF`).  
2. Implement a `type MyDevice struct{ … }` with methods to handle reads/writes.  
3. Extend the **Bus** in `bus.go`:
   - In `Read8/Write8`, check if the address hits your device range; delegate.  
4. Document the registers in a small comment table.

---

## Example: the UART registers

| Address        | Name          | Direction | Meaning                         |
|----------------|---------------|-----------|---------------------------------|
| `0x1000_0000`  | `UART_TX`     | write     | Write a byte ⇒ prints to stdout |
| `0x1000_0004`  | `UART_STATUS` | read      | Always returns `1` (ready)      |

Writing a **word** to `UART_TX` prints the **low byte** too (convenience for simple programs).

---

## Unit tests (what they cover)

`sim/cpu_test.go` includes three tiny tests that don’t require a toolchain:

- **UART + ECALL:** program writes `'A'` to UART then halts; test captures stdout.  
- **LB sign-extension:** loads `0xFF` and expects `0xFFFFFFFF`.  
- **BEQ branching:** verifies a taken branch skips over an instruction.

Run them with:

```bash
go test ./...
```

---

## Troubleshooting

- **“No program provided. Use -elf or -bin.”**  
  Pass `-elf examples/uart_hello/uart_hello.elf` **or** `-bin examples/uart_hello/uart_hello.bin`.

- **Nothing prints / garbled output**  
  Ensure your program writes to `0x1000_0000` (byte or word). For bytes, `SB` is ideal.

- **Out-of-bounds or “trap” messages**  
  Your program accessed RAM/MMIO outside mapped ranges. Check your linker script and addresses.

- **Toolchain prefix mismatch**  
  Use `RISCV_PREFIX=riscv32-unknown-elf make` if your compiler is `riscv32-unknown-elf-gcc`.

---

## Tips for teaching

- Turn on `-trace` for live demos; turn it off to show runtime effects only.  
- Keep **each new feature** in its own commit so students can diff.  
- Encourage students to write a **1–2 instruction unit test** for each opcode they add.

---

## License

Educational use; choose and add your preferred license file (e.g., MIT) if you plan to share.
