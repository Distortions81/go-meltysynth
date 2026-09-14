package meltysynth

import (
	"math"
	"testing"
)

func TestCentsLookupAccuracy(t *testing.T) {
	for cents := float32(-24000); cents <= 24000; cents += 0.25 {
		got := calcCentsToMultiplyingFactor(cents)
		want := float32(math.Exp2(float64(cents) / 1200))
		assertRelativeError(t, "cents", cents, got, want, 2.0e-6)
	}
}

func TestDecibelsLookupAccuracy(t *testing.T) {
	for decibels := float32(-200); decibels <= 100; decibels += 0.01 {
		got := calcDecibelsToLinear(decibels)
		want := float32(math.Pow(10, 0.05*float64(decibels)))
		assertRelativeError(t, "decibels", decibels, got, want, 1.0e-5)
	}
}

func TestLookupFallbacks(t *testing.T) {
	for _, cents := range []float32{-30000, 30000} {
		got := calcCentsToMultiplyingFactor(cents)
		want := float32(math.Exp2(float64(cents) / 1200))
		if got != want {
			t.Fatalf("cents fallback at %v: got %v, want %v", cents, got, want)
		}
	}

	for _, decibels := range []float32{-300, 200} {
		got := calcDecibelsToLinear(decibels)
		want := float32(math.Pow(10, 0.05*float64(decibels)))
		if got != want {
			t.Fatalf("decibels fallback at %v: got %v, want %v", decibels, got, want)
		}
	}

	if got := calcCentsToMultiplyingFactor(float32(math.NaN())); !math.IsNaN(float64(got)) {
		t.Fatalf("cents NaN fallback returned %v", got)
	}
	if got := calcDecibelsToLinear(float32(math.NaN())); !math.IsNaN(float64(got)) {
		t.Fatalf("decibels NaN fallback returned %v", got)
	}
}

func assertRelativeError(t *testing.T, name string, input, got, want, limit float32) {
	t.Helper()
	error := float32(math.Abs(float64((got - want) / want)))
	if error > limit {
		t.Fatalf("%s lookup at %v: got %v, want %v, relative error %v exceeds %v", name, input, got, want, error, limit)
	}
}
