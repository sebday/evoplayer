package viz

import "math"

func fftInPlace(re, im []float64) {
	n := len(re)
	if n < 2 || n&(n-1) != 0 {
		return
	}
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for j&bit != 0 {
			j ^= bit
			bit >>= 1
		}
		j ^= bit
		if i < j {
			re[i], re[j] = re[j], re[i]
			im[i], im[j] = im[j], im[i]
		}
	}
	for length := 2; length <= n; length <<= 1 {
		angle := -2 * math.Pi / float64(length)
		wLenRe := math.Cos(angle)
		wLenIm := math.Sin(angle)
		for i := 0; i < n; i += length {
			wRe, wIm := 1.0, 0.0
			half := length >> 1
			for j := 0; j < half; j++ {
				uRe := re[i+j]
				uIm := im[i+j]
				vRe := re[i+j+half]*wRe - im[i+j+half]*wIm
				vIm := re[i+j+half]*wIm + im[i+j+half]*wRe
				re[i+j] = uRe + vRe
				im[i+j] = uIm + vIm
				re[i+j+half] = uRe - vRe
				im[i+j+half] = uIm - vIm
				nextRe := wRe*wLenRe - wIm*wLenIm
				wIm = wRe*wLenIm + wIm*wLenRe
				wRe = nextRe
			}
		}
	}
}
