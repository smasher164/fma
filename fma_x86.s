// +build 386 amd64 amd64p32

#include "textflag.h"

TEXT ·isFMASupported(SB),NOSPLIT,$0
	MOVB $1, AL
	CPUID
	// AND CX, $(1<<12)
	BYTE $0x66; BYTE $0x81; BYTE $0xE1; BYTE $0x00; BYTE $0x10
	MOVB CH, ret(FP)
	RET

// func FMA(x, y, z float64) float64
TEXT ·FMA(SB),NOSPLIT,$0
	MOVB ·hasFMA(SB), AL
	CMPB AL, $0
	JE soft
	// hardware supports FMA3
	MOVLPD x+0(FP), X0
	MOVLPD y+8(FP), X1
	MOVLPD z+16(FP), X2
	// X0 = X0 * X1 + X2
	VFMADD213SD X2, X1, X0
	MOVLPD X0, ret+24(FP)
	RET
soft:
	JMP ·fma(SB)
