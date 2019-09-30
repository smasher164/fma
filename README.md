# fma

This repository contains a software implementation of Fused-Multiply Add (FMA) for 64-bit floating-point values written in Go. The operation conforms to IEEE-754, but only offers the round-to-nearest ties-to-even rounding mode. This implementation is intended to be used in the Go standard library following [CL 127458](https://go-review.googlesource.com/c/go/+/127458).