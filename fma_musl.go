// MIT License

// Copyright (c) 2018 Akhil Indurti
// Copyright (c) 2005-2014 Rich Felker, et al.

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

import (
	"math"
	"math/bits"
)

// mul64 returns the most significant and least significant
// 64 bits from the 128-bit product of x and y.
func mul64(x, y uint64) (hi, lo uint64) {
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

	lo = (t << 32) + w0
	hi = xhi*yhi + w2 + k
	return
}

type num struct {
	m    uint64
	e    int32
	sign int32
}

func normalize(x float64) num {
	ix := math.Float64bits(x)
	e := int32(ix >> 52)
	sign := e & 0x800
	e &= 0x7ff
	if e == 0 {
		ix = math.Float64bits(x * (1 << 63))
		e = int32(ix >> 52 & 0x7ff)
		if e == 0 {
			e = 0x800
		} else {
			e -= 63
		}
	}
	ix &= (1 << 52) - 1
	ix |= 1 << 52
	ix <<= 1
	e -= 0x3ff + 52 + 1
	return num{ix, e, sign}
}

// FMA_MUSL is a portable implementation of the floating-point
// multiply-add operation (x * y + z) with mostly integer
// arithmetic. The result's precision is guaranteed to match
// that of the FMA operation in IEEE 754-2008.
func FMA_MUSL(x, y, z float64) float64 {
	// normalize so top 10 bits and last bit are 0
	nx := normalize(x)
	ny := normalize(y)
	nz := normalize(z)

	const ZEROINFNAN = 0x7ff - 0x3ff - 52 - 1
	if nx.e >= ZEROINFNAN || ny.e >= ZEROINFNAN {
		return x*y + z
	}
	if nz.e >= ZEROINFNAN {
		if nz.e > ZEROINFNAN {
			return x*y + z
		}
		return z
	}

	// r = x * y
	rhi, rlo := mul64(nx.m, ny.m)
	// align exponents
	e := nx.e + ny.e
	d := nz.e - e
	var zlo, zhi uint64
	// shift bits z<<=kz, r>>=kr, so kz+kr == d, set e = e+kr (== ez-kz)
	if d > 0 {
		if d < 64 {
			zlo = nz.m << uint(d)
			zhi = nz.m >> (64 - uint(d))
		} else {
			zhi = nz.m
			e = nz.e - 64
			d -= 64
			if d < 64 {
				rlo = rhi<<(64-uint(d)) | rlo>>uint(d)
				if (rlo << (64 - uint(d))) != 0 {
					rlo |= 1
				}
				rhi = rhi >> uint(d)
			} else if d != 0 {
				rlo = 1
				rhi = 0
			}
		}
	} else {
		d = -d
		if d == 0 {
			zlo = nz.m
		} else if d < 64 {
			zlo = nz.m >> uint(d)
			if (nz.m << (64 - uint(d))) != 0 {
				zlo |= 1
			}
		} else {
			zlo = 1
		}
	}

	// add
	sign32 := (nx.sign ^ ny.sign)
	sign := sign32 != 0
	samesign := (sign32 ^ nz.sign) == 0
	nonzero := true
	if samesign {
		rlo += zlo
		rhi += zhi
		if rlo < zlo {
			rhi++
		}
	} else {
		t := rlo
		rlo -= zlo
		rhi = rhi - zhi
		if t < rlo {
			rhi--
		}
		if (rhi >> 63) != 0 {
			rlo = -rlo
			rhi = -rhi
			if rlo != 0 {
				rhi--
			}
			sign = !sign
		}
		nonzero = rhi != 0
	}

	// set rhi to top 63bit of the result (last bit is sticky)
	if nonzero {
		e += 64
		d = int32(bits.LeadingZeros64(rhi) - 1)
		rhi = rhi<<uint(d) | rlo>>(64-uint(d))
		if (rlo << uint(d)) != 0 {
			rhi |= 1
		}
	} else if rlo != 0 {
		d = int32(bits.LeadingZeros64(rlo) - 1)
		if d < 0 {
			rhi = rlo>>1 | (rlo & 1)
		} else {
			rhi = rlo << uint(d)
		}
	} else {
		// exact +-0
		return x*y + z
	}
	e -= d

	// convert to float64
	// i is in [1<<62,(1<<63)-1]
	i := int64(rhi)
	if sign {
		i = -i
	}
	// |r| is in [0x1p62,0x1p63]
	r := float64(i)

	if e < -1022-62 {
		// result is subnormal before rounding
		if e == -1022-63 {
			const FLT_MIN = 1.1754943508222875079687365372222456778186655567720875e-38
			const DBL_MIN = 2.225073858507201383090232717332404064219215980462331e-308
			c := float64(1 << 63)
			if sign {
				c = -c
			}
			if r == c {
				// min normal after rounding, underflow depends
				// on arch behaviour which can be imitated by
				// a double to float conversion
				fltmin := float32((0.268435448 * (1.0 / (1 << 63))) * FLT_MIN * r)
				return float64(DBL_MIN / FLT_MIN * fltmin)
			}
			// one bit is lost when scaled, add another top bit to
			// only round once at conversion if it is inexact
			if (rhi << 53) != 0 {
				i = int64(rhi>>1 | (rhi & 1) | 1<<62)
				if sign {
					i = -i
				}
				r = float64(i)
				// remove top bit
				r = 2*r - c
				{
					// raise underflow portably, such that it
					// cannot be optimized away
					tiny := DBL_MIN / FLT_MIN * r
					r += (tiny * tiny) * (r - r)
				}
			}
		} else {
			// only round once when scaled
			d = 10
			ui := rhi >> uint(d)
			if (rhi << (64 - uint(d))) != 0 {
				ui |= 1
			}
			i = int64(ui << uint(d))
			if sign {
				i = -i
			}
			r = float64(i)
		}
	}
	return math.Ldexp(r, int(e))
}
