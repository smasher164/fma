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

/*-
 * SPDX-License-Identifier: BSD-2-Clause-FreeBSD
 *
 * Copyright (c) 2005-2011 David Schultz <das@FreeBSD.ORG>
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions
 * are met:
 * 1. Redistributions of source code must retain the above copyright
 *    notice, this list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright
 *    notice, this list of conditions and the following disclaimer in the
 *    documentation and/or other materials provided with the distribution.
 *
 * THIS SOFTWARE IS PROVIDED BY THE AUTHOR AND CONTRIBUTORS ``AS IS'' AND
 * ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED.  IN NO EVENT SHALL THE AUTHOR OR CONTRIBUTORS BE LIABLE
 * FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
 * DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS
 * OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION)
 * HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT
 * LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY
 * OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF
 * SUCH DAMAGE.
 */

package fma

import (
	"math"
)

type f128 struct{ hi, lo float64 }

func addf128(a, b float64) f128 {
	hi := a + b
	s := hi - a
	lo := (a - (hi - s)) + (b - s)
	return f128{hi, lo}
}

func mulf128(a, b float64) f128 {
	const split = 1<<27 + 1.0
	p := a * split
	ha := a - p
	ha += p
	la := a - ha

	p = b * split
	hb := b - p
	hb += p
	lb := b - hb

	p = ha * hb
	q := ha*lb + la*hb
	hi := p + q
	lo := p - hi + q + la*lb
	return f128{hi, lo}
}

func addAdjusted(a, b float64) float64 {
	sum := addf128(a, b)
	if sum.lo != 0 {
		uhi := math.Float64bits(sum.hi)
		if (uhi & 1) == 0 {
			ulo := math.Float64bits(sum.lo)
			uhi += 1 - ((uhi ^ ulo) >> 62)
			sum.hi = math.Float64frombits(uhi)
		}
	}
	return sum.hi
}

func addAndDenormalize(a, b float64, scale int) float64 {
	sum := addf128(a, b)
	if sum.lo != 0 {
		uhi := math.Float64bits(sum.hi)
		bitsLost := -(int(uhi>>52) & 0x7FF) - scale + 1
		pred := 0
		if bitsLost != 1 {
			pred = 1
		}
		if (pred ^ int(uhi&1)) != 0 {
			ulo := math.Float64bits(sum.lo)
			uhi += 1 - (((uhi ^ ulo) >> 62) & 2)
			sum.hi = math.Float64frombits(uhi)
		}
	}
	return math.Ldexp(sum.hi, scale)
}

func isfinite(x float64) bool {
	return !math.IsInf(x, 0) && !math.IsNaN(x)
}

// FMA_BSD is a portable implementation of the floating-point
// multiply-add operation (x * y + z) with mostly float arithmetic.
// The result's precision is guaranteed to match
// that of the FMA operation in IEEE 754-2008.
func FMA_BSD(x, y, z float64) float64 {
	if x == 0.0 || y == 0.0 {
		return x*y + z
	}
	if z == 0.0 {
		return x * y
	}
	if !isfinite(x) || !isfinite(y) {
		return x*y + z
	}
	if !isfinite(z) {
		return z
	}
	xs, ex := math.Frexp(x)
	ys, ey := math.Frexp(y)
	zs, ez := math.Frexp(z)
	spread := ex + ey - ez

	const DBL_MANT_DIG = 53
	if spread < -DBL_MANT_DIG {
		return z
	}
	if spread <= DBL_MANT_DIG*2 {
		zs = math.Ldexp(zs, -spread)
	} else {
		const DBL_MIN = 2.225073858507201383090232717332404064219215980462331e-308
		zs = math.Copysign(DBL_MIN, zs)
	}
	xy := mulf128(xs, ys)
	r := addf128(xy.hi, zs)
	spread = ex + ey

	if r.hi == 0.0 {
		return xy.hi + zs + math.Ldexp(xy.lo, spread)
	}
	adj := addAdjusted(r.lo, xy.lo)
	if spread+math.Ilogb(r.hi) > -1023 {
		return math.Ldexp(r.hi+adj, spread)
	}
	return addAndDenormalize(r.hi, adj, spread)
}
