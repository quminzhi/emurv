ENTRY(_start)

SECTIONS
{
  /* Put code/data at 0x00000000 so our simple emulator can run it */
  . = 0x00000000;

  .text : {
    *(.text .text.*)
  }

  .rodata : {
    *(.rodata .rodata.*)
  }

  .data : {
    *(.data .data.*)
  }

  .bss : {
    *(.bss .bss.*)
    *(COMMON)
  }
}
