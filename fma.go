package fma

func FMA(x, y, z float64) float64
func fma(x, y, z float64) float64 {
	// return FMA_BSD(x, y, z)
	return FMA_MUSL(x, y, z)
}
