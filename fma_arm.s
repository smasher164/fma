#include "textflag.h"

// func FMA(x, y, z float64) float64
TEXT ·FMA(SB),NOSPLIT,$0
	MOVB ·hasFMA(SB), R0
	CMP $0, R0
	BEQ soft
	// The following won't compile without WORD directives.
	// It's okay, since we probed for the instruction first.
	WORD $0XFD4007E0 	// FMOVD x+0(FP), F0
	WORD $0XFD400BE1 	// FMOVD y+8(FP), F1
	WORD $0XFD400FE2 	// FMOVD z+16(FP), F2
	// F0 = F0 * F1 + F2
	WORD $0X1F410800 	// FMADD D0, D0, D1, D2
	WORD $0XFD0013E0 	// FMOVD F0, ret+24(FP)
	RET
soft:
	B ·fma(SB)

TEXT ·probe(SB),NOSPLIT,$0
	WORD $0x1F410800
