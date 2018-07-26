#include "textflag.h"

TEXT ·isFMASupported(SB),NOSPLIT,$0
	MOVW R0, R2
#ifndef GOMIPS_softfloat
	// Detect Release 6. ADDI < R6 == BOVC on R6.
	// See https://github.com/v8mips/v8mips/issues/97#issue-44761752
	WORD $0x20420001
	BNE R0, R2, nosupport
	// Detect double-precision. CP1.FIR[18:17] == 1
	MOVW FCR0, R2
	MOVW $(1<<17), R9
	AND R9, R2, R2
	SRL $17, R2, R2
nosupport:
#endif
	MOVW R2, ret(FP)
	RET

// func FMA(x, y, z float64) float64
TEXT ·FMA(SB),NOSPLIT,$0
	MOVB ·hasFMA(SB), R8
	BEQ R0, R8, soft
	// hardware supports fma
	MOVD x+0(FP), F0
	MOVD y+8(FP), F1
	MOVD z+16(FP), F2
	// F2 = F2 + F0 * F1
	// MADDF.D F2, F0, F1
	WORD $0x46210098
	MOVD F2, ret+24(FP)
	RET
soft:
	JMP ·fma(SB)
