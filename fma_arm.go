package fma

var hasFMA bool

func init() {
	hasFMA = isFMASupported()
}

// probe issues an FMA instruction, which should cause a
// SIGILL on POSIX systems that don't support hardware FMA.
func probe() bool

// recover from SIGILL to determine FMA support.
func isFMASupported() bool {
	v := true
	func() {
		defer func() {
			if r := recover(); r != nil {
				v = false
			}
		}()
		probe()
	}()
	return v
}
