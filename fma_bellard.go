// MIT License

// Copyright (c) 2018 Akhil Indurti

// Permission is hereby granted, free of charge, to any person obtaining
// a copy of this software and associated documentation files (the
// "Software"), to deal in the Software without restriction, including
// without limitation the rights to use, copy, modify, merge, publish,
// distribute, sublicense, and/or sell copies of the Software, and to
// permit persons to whom the Software is furnished to do so, subject to
// the following conditions:

// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
// IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
// CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
// TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
// SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package fma

// The original implementation is written by Fabrice Bellard for SoftFP and can
// be found at https://bellard.org/softfp/. The Go code is a simplified version
// of the original C, notably only rounding ties to even.
// SoftFP is licensed as follows:
//
// SoftFP Library
//
// Copyright (c) 2016 Fabrice Bellard
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL
// THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

import (
	"math"
	"math/bits"
)

const (
	EXP_SIZE   = 11
	EXP_MASK   = (1 << EXP_SIZE) - 1
	MANT_SIZE  = 52
	MANT_MASK  = (1 << MANT_SIZE) - 1
	IMANT_SIZE = 62
	RND_SIZE   = IMANT_SIZE - MANT_SIZE
)

func isnan(x uint64) bool {
	return (x>>MANT_SIZE)&EXP_MASK == EXP_MASK && (x&MANT_MASK) != 0
}

func pack(sign, exp uint32, mant uint64) uint64 {
	return (uint64(sign) << (64 - 1)) | (uint64(exp) << MANT_SIZE) | (mant & MANT_MASK)
}

func normalizeSubnormal(pexp *int32, mant uint64) uint64 {
	shift := int32(MANT_SIZE - (64 - 1 - bits.LeadingZeros64(mant)))
	*pexp = 1 - shift
	return mant << uint(shift)
}

func umul(plo *uint64, x, y uint64) (hi uint64) {
	// See http://www.hackersdelight.org/MontgomeryMultiplication.pdf
	//
	// Montgomery Multiplication for fixed-width values:
	// Extract the higher and lower halves of x and y.
	// Product = x*y*r mod m, calculated more efficiently
	// with just three multiplications.

	xlo := x & 0xFFFFFFFF
	xhi := x >> 32
	ylo := y & 0xFFFFFFFF
	yhi := y >> 32

	t := xlo * ylo
	w0 := t & 0xFFFFFFFF
	k := t >> 32

	t = xhi*ylo + k
	w1 := t & 0xFFFFFFFF
	w2 := t >> 32

	t = xlo*yhi + w1
	k = t >> 32

	*plo = (t << 32) + w0
	hi = xhi*yhi + w2 + k
	return
}

func normalize_bellard(asign uint32, aexp int32, amant uint64) uint64 {
	shift := int32(bits.LeadingZeros64(amant) - (64 - 1 - IMANT_SIZE))
	aexp -= shift
	amant <<= uint(shift)
	return roundPack(asign, aexp, amant)
}

func normalize2(asign uint32, aexp int32, amant1, amant0 uint64) uint64 {
	var l int32
	if amant1 == 0 {
		l = int32(64 + bits.LeadingZeros64(amant0))
	} else {
		l = int32(bits.LeadingZeros64(amant1))
	}
	l -= (64 - 1 - IMANT_SIZE)
	shift := uint(l)
	aexp -= l
	if shift == 0 {
		if amant0 != 0 {
			amant1 |= 1
		}
	} else if shift < 64 {
		amant1 = (amant1 << shift) | (amant0 >> (64 - shift))
		amant0 <<= shift
		if amant0 != 0 {
			amant1 |= 1
		}
	} else {
		amant1 = amant0 << (shift - 64)
	}
	return roundPack(asign, aexp, amant1)
}

func roundPack(asign uint32, aexp int32, amant uint64) uint64 {
	const addend = 1 << (RND_SIZE - 1)
	if aexp <= 0 {
		diff := 1 - aexp
		amant = rshiftRnd(amant, diff)
		aexp = 1
	}
	rndBits := uint32(amant & ((1 << RND_SIZE) - 1))
	amant = (amant + addend) >> RND_SIZE
	if rndBits == addend {
		n := int64(^1)
		amant &= uint64(n)
	}
	aexp += int32(amant >> (MANT_SIZE + 1))
	if amant <= MANT_MASK {
		aexp = 0
	} else if aexp >= EXP_MASK {
		aexp = EXP_MASK
		amant = 0
	}
	return pack(asign, uint32(aexp), amant)
}

func rshiftRnd(a uint64, d int32) uint64 {
	if d != 0 {
		if d >= 64 {
			if a != 0 {
				a = 1
			}
		} else {
			ud := uint64(d)
			mask := uint64((1 << ud) - 1)
			a >>= ud
			if (a & mask) != 0 {
				a |= 1
			}
		}
	}
	return a
}

func fma_bellard(a, b, c uint64) uint64 {
	const QNAN = (EXP_MASK << MANT_SIZE) | (1 << (MANT_SIZE - 1))
	asign := uint32(a >> 63)
	bsign := uint32(b >> 63)
	csign := uint32(c >> 63)
	rsign := asign ^ bsign
	aexp := int32((a >> MANT_SIZE) & EXP_MASK)
	bexp := int32((b >> MANT_SIZE) & EXP_MASK)
	cexp := int32((c >> MANT_SIZE) & EXP_MASK)
	amant := a & MANT_MASK
	bmant := b & MANT_MASK
	cmant := c & MANT_MASK
	if aexp == EXP_MASK || bexp == EXP_MASK || cexp == EXP_MASK {
		if isnan(a) || isnan(b) || isnan(c) {
			return QNAN
		} else {
			// infinities
			if (aexp == EXP_MASK && (bexp == 0 && bmant == 0)) ||
				(bexp == EXP_MASK && (aexp == 0 && amant == 0)) ||
				((aexp == EXP_MASK || bexp == EXP_MASK) &&
					(cexp == EXP_MASK && rsign != csign)) {
				return QNAN
			} else if cexp == EXP_MASK {
				return pack(csign, EXP_MASK, 0)
			} else {
				return pack(rsign, EXP_MASK, 0)
			}
		}
	}
	if aexp == 0 {
		if amant == 0 {
			if cexp == 0 && cmant == 0 {
				return pack(0, 0, 0)
			} else {
				return c
			}
		}
		amant = normalizeSubnormal(&aexp, amant)
	} else {
		amant |= 1 << MANT_SIZE
	}
	if bexp == 0 {
		if bmant == 0 {
			if cexp == 0 && cmant == 0 {
				return pack(0, 0, 0)
			} else {
				return c
			}
		}
		bmant = normalizeSubnormal(&bexp, bmant)
	} else {
		bmant |= 1 << MANT_SIZE
	}
	// multiply
	rexp := aexp + bexp - (1 << (EXP_SIZE - 1)) + 3

	var rmant0 uint64
	rmant1 := umul(&rmant0, amant<<RND_SIZE, bmant<<RND_SIZE)
	// normalize to 64-3
	if rmant1 < (1 << 61) {
		rmant1 = (rmant1 << 1) | (rmant0 >> 63)
		rmant0 <<= 1
		rexp--
	}

	// add
	if cexp == 0 {
		if cmant == 0 {
			// add zero
			if rmant0 != 0 {
				rmant1 |= 1
			}
			return normalize_bellard(rsign, rexp, rmant1)
		}
		cmant = normalizeSubnormal(&cexp, cmant)
	} else {
		cmant |= (1 << MANT_SIZE)
	}
	cexp++
	cmant1 := cmant << (RND_SIZE - 1)
	cmant0 := uint64(0)

	// ensure that abs(r) >= abs(c)
	if !(rexp > cexp || (rexp == cexp && rmant1 >= cmant1)) {
		rmant1, cmant1 = cmant1, rmant1
		rmant0, cmant0 = cmant0, rmant0
		rexp, cexp = cexp, rexp
		rsign, csign = csign, rsign
	}
	// right shift cmant
	shift := uint(rexp - cexp)
	if shift >= 128 {
		if (cmant0 | cmant1) != 0 {
			cmant0 |= 1
		}
		cmant1 = 0
	} else if shift >= 65 {
		cmant0 = rshiftRnd(cmant1, int32(shift)-64)
		cmant1 = 0
	} else if shift == 64 {
		if cmant0 != 0 {
			cmant0 |= 1
		}
		cmant0 |= cmant1
		cmant1 = 0
	} else if shift != 0 {
		mask := uint64((1 << shift) - 1)
		c := cmant0
		cmant0 |= (cmant1 << (64 - shift)) | (cmant0 >> shift)
		if (c & mask) != 0 {
			cmant0 |= 1
		}
		cmant1 >>= shift
	}

	// add or subtract
	if rsign == csign {
		rmant0 += cmant0
		rmant1 += cmant1
		if rmant0 < cmant0 {
			rmant1 |= 1
		}
	} else {
		tmp := rmant0
		rmant0 -= cmant0
		rmant1 = rmant1 - cmant1
		if rmant0 > tmp {
			rmant1--
		}
		if (rmant0 | rmant1) == 0 {
			rsign = 0
		}
	}
	return normalize2(rsign, rexp, rmant1, rmant0)
}

// FMA_Bellard is a portable implementation of the floating-point
// multiply-add operation (x * y + z) where the result's precision
// is guaranteed to match that of the FMA operation in IEEE 754-2008.
func FMA_Bellard(x, y, z float64) float64 {
	a := math.Float64bits(x)
	b := math.Float64bits(y)
	c := math.Float64bits(z)
	r := fma_bellard(a, b, c)
	return math.Float64frombits(r)
}
