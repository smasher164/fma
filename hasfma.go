// +build 386 amd64 amd64p32 mips mipsle mips64 mips64le

package fma

var hasFMA bool

func init() {
	hasFMA = isFMASupported()
}

func isFMASupported() bool
