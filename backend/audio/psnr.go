// Package audio is made to handle psnr for audios
package audio

import (
	"math"
)

func CalculatePSNR(original, stego []byte) float64 {
	if len(original) != len(stego) {
		return 0.0
	}

	if len(original) == 0 {
		return 0.0
	}

	var mse float64
	for i := range original {
		diff := float64(original[i]) - float64(stego[i])
		mse += diff * diff
	}
	mse /= float64(len(original))

	// If MSE is 0, signals are identical
	if mse == 0 {
		return math.Inf(1)
	}

	// Calculate PSNR in dB
	// PSNR = 20 * log10(MAX_SIGNAL_VALUE / sqrt(MSE))
	// For 8-bit audio, MAX_SIGNAL_VALUE = 255
	maxSignalValue := 255.0
	psnr := 20 * math.Log10(maxSignalValue/math.Sqrt(mse))

	return psnr
}

// CalculatePSNRFloat64 calculates PSNR for float64 audio samples
func CalculatePSNRFloat64(original, stego []float64) float64 {
	if len(original) != len(stego) {
		return 0.0
	}

	if len(original) == 0 {
		return 0.0
	}

	// Calculate Mean Squared Error (MSE)
	var mse float64
	for i := range original {
		diff := original[i] - stego[i]
		mse += diff * diff
	}
	mse /= float64(len(original))

	// If MSE is 0, signals are identical
	if mse == 0 {
		return math.Inf(1) // Infinite PSNR
	}

	// Calculate PSNR in dB
	// For normalized float audio (-1.0 to 1.0), MAX_SIGNAL_VALUE = 1.0
	maxSignalValue := 1.0
	psnr := 20 * math.Log10(maxSignalValue/math.Sqrt(mse))

	return psnr
}

func ValidatePSNR(psnr float64, threshold float64) bool {
	if math.IsInf(psnr, 1) {
		return true // Infinite PSNR is always good
	}
	return psnr >= threshold
}
