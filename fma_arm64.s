#include "textflag.h"

// func FMA(x, y, z float64) float64
TEXT ·FMA(SB),NOSPLIT,$0
	FMOVD x+0(FP), F0
	FMOVD y+8(FP), F1
	FMOVD z+16(FP), F2
	FMADDD F0, F2, F1, F0
	FMOVD F0, ret+24(FP)
	RET
