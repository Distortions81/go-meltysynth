package meltysynth

import (
	"math"
)

const (
	halfPi                    float32 = math.Pi / 2
	nonAudible                float32 = 1.0e-3
	centsLookupMin            float32 = -24000
	centsLookupMax            float32 = 24000
	centsLookupStep           float32 = 4
	centsLookupInverseStep    float32 = 1 / centsLookupStep
	centsLookupSize                   = int((centsLookupMax-centsLookupMin)/centsLookupStep) + 1
	decibelsLookupMin         float32 = -200
	decibelsLookupMax         float32 = 100
	decibelsLookupStep        float32 = 0.05
	decibelsLookupInverseStep float32 = 1 / decibelsLookupStep
	decibelsLookupSize                = int((decibelsLookupMax-decibelsLookupMin)/decibelsLookupStep) + 1
	naturalLogToDecibels      float64 = 8.685889638065037
)

var logNonAudible float32 = float32(math.Log(1.0e-3))

var centsLookup [centsLookupSize]float32
var decibelsLookup [decibelsLookupSize]float32

func init() {
	for i := range centsLookup {
		cents := centsLookupMin + float32(i)*centsLookupStep
		centsLookup[i] = float32(math.Exp2(float64(cents) / 1200))
	}
	for i := range decibelsLookup {
		decibels := decibelsLookupMin + float32(i)*decibelsLookupStep
		decibelsLookup[i] = float32(math.Pow(10, 0.05*float64(decibels)))
	}
}

func calcTimecentsToSeconds(x float32) float32 {
	return calcCentsToMultiplyingFactor(x)
}

func calcCentsToHertz(x float32) float32 {
	return 8.176 * calcCentsToMultiplyingFactor(x)
}

func calcCentsToMultiplyingFactor(x float32) float32 {
	if x != x || x < centsLookupMin || x > centsLookupMax {
		return float32(math.Exp2(float64(x) / 1200))
	}

	position := (x - centsLookupMin) * centsLookupInverseStep
	index := int(position)
	if index >= len(centsLookup)-1 {
		return centsLookup[len(centsLookup)-1]
	}
	fraction := position - float32(index)
	value := centsLookup[index]
	return value + fraction*(centsLookup[index+1]-value)
}

func calcDecibelsToLinear(x float32) float32 {
	if x != x || x < decibelsLookupMin || x > decibelsLookupMax {
		return float32(math.Pow(10, 0.05*float64(x)))
	}

	position := (x - decibelsLookupMin) * decibelsLookupInverseStep
	index := int(position)
	if index >= len(decibelsLookup)-1 {
		return decibelsLookup[len(decibelsLookup)-1]
	}
	fraction := position - float32(index)
	value := decibelsLookup[index]
	return value + fraction*(decibelsLookup[index+1]-value)
}

func calcLinearToDecibels(x float32) float32 {
	return float32(float64(20) * math.Log10(float64(x)))
}

func calcKeyNumberToMultiplyingFactor(cents int32, key int32) float32 {
	return calcTimecentsToSeconds(float32(cents * (60 - key)))
}

func calcExpCutoff(x float64) float64 {
	if x < float64(logNonAudible) {
		return 0
	} else {
		return float64(calcDecibelsToLinear(float32(naturalLogToDecibels * x)))
	}
}

func calcClamp(value float32, min float32, max float32) float32 {
	switch {
	case value < min:
		return min
	case value > max:
		return max
	default:
		return value
	}
}
