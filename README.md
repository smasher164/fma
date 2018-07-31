# fma

This repository contains two software implementations for Fused-Multiply Add (FMA) for 64-bit floating-point values written in Go. The goal is to have a precision guaranteed to match that of hardware implementations of the IEEE-754 2008 standard's FMA operation.
- The first is a translation of [MUSL](http://git.musl-libc.org/cgit/musl/tree/src/math/fma.c)'s implementation that uses mostly integer arithmetic.
- The second is a translation of [FreeBSD](https://svnweb.freebsd.org/base/head/lib/msun/src/s_fma.c?view=markup)'s implementation that uses mostly floating-point arithmetic.

The tests are generated by [Berkeley TestFloat 3e](http://www.jhauser.us/arithmetic/TestFloat.html), which verifies conformity to the IEEE Standard.

Work still needs to be done to include:
- [x] Benchmarks to compare and improve performance of the two implementations.
- [ ] Assembly implementations that use the above software implementation as a fallback.
	- [x] 386, amd64, amd64p32
	- [x] arm64
	- [ ] arm
		- [ ] Feature Detection
	- [ ] mipsx, mips64x
		- [ ] Feature Detection
	- [x] ppc64x
	- [x] s390x