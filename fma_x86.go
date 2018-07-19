// +build 386 amd64 amd64p32

package fma

// cache result so that CPUID isn't issued on every call to FMA
var hasFMA bool

func init() {
	hasFMA = isFMASupported()
}

// stub. see fma_x86.s
func isFMASupported() bool
