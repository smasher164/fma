#include "textflag.h"

// func FMA(x, y, z float64) float64
TEXT ·FMA(SB),NOSPLIT,$0
	B ·fma(SB)
