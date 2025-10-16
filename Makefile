RISCV_PREFIX ?= riscv64-unknown-elf
AS     = $(RISCV_PREFIX)-gcc
OBJCOPY= $(RISCV_PREFIX)-objcopy

CFLAGS = -nostdlib -march=rv32i -mabi=ilp32 -Wl,-T,user/uart_hello/linker.ld

all: user/uart_hello/uart_hello.elf user/uart_hello/uart_hello.bin

test:
	go test ./...

user/uart_hello/uart_hello.elf: user/uart_hello/hello.s user/uart_hello/linker.ld
	$(AS) $(CFLAGS) -o $@ $<

user/uart_hello/uart_hello.bin: user/uart_hello/uart_hello.elf
	$(OBJCOPY) -O binary $< $@

run-elf: all
	go run ./cmd/emurv -elf user/uart_hello/uart_hello.elf -trace

run-bin: all
	go run ./cmd/emurv -bin user/uart_hello/uart_hello.bin -trace

clean:
	rm -f user/uart_hello/uart_hello.elf user/uart_hello/uart_hello.bin
