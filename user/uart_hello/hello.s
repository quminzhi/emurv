    .section .text
    .globl _start
_start:
    la   t0, msg           # t0 = &msg
    li   t1, 0x10000000    # UART_TX (MMIO)
1:
    lbu  t2, 0(t0)         # load byte
    beq  t2, x0, 2f        # if zero -> end
    sb   t2, 0(t1)         # store byte to UART TX
    addi t0, t0, 1
    j    1b
2:
    ecall                  # halt in our emulator

    .section .rodata
msg:
    .ascii "Hello, RISC-V!\\n"
    .byte 0
