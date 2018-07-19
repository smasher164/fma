// +build 386 amd64 amd64p32

#include "textflag.h"

TEXT ·isFMASupported(SB),NOSPLIT,$0
	MOVB $1, AL
	CPUID
	// SHR CX, $12
	BYTE $0x66; BYTE $0xc1; BYTE $0xe9; BYTE $0x0c
	// AND CL, $1
	BYTE $0x80; BYTE $0xe1; BYTE $0x01
	MOVB CL, ret(FP)
	RET

// func FMA(x, y, z float64) float64
TEXT ·FMA(SB),NOSPLIT,$0
	MOVB ·hasFMA(SB), AL
	CMPB AL, $0
	JE soft
	// if hardware supports FMA3
	MOVLPD x+0(FP), X0
	MOVLPD y+8(FP), X1
	MOVLPD z+16(FP), X2
	// X0 = X0 * X1 + X2
	VFMADD213SD X2, X1, X0
	MOVLPD X0, ret+24(FP)
	RET
soft:
	JMP ·fma(SB)
