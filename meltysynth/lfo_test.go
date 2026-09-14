package meltysynth

import (
	"math"
	"testing"
)

func TestLfoPhaseAccumulatorMatchesTimeBasedWaveform(t *testing.T) {
	settings := NewSynthesizerSettings(44100)
	synth := &Synthesizer{SampleRate: settings.SampleRate, BlockSize: settings.BlockSize}

	for _, test := range []struct {
		delay     float32
		frequency float32
		blocks    int
	}{
		{delay: 0, frequency: 5.25, blocks: 10000},
		{delay: 0.125, frequency: 6.1, blocks: 10000},
		{delay: 0, frequency: 1000, blocks: 100},
	} {
		var oscillator lfo
		oscillator.synthesizer = synth
		oscillator.start(test.delay, test.frequency)
		for blockIndex := 1; blockIndex <= test.blocks; blockIndex++ {
			oscillator.process()
			want := referenceLfoValue(blockIndex, synth, test.delay, test.frequency)
			if difference := math.Abs(float64(oscillator.value - want)); difference > 1.0e-5 {
				t.Fatalf("delay %v frequency %v block %d: got %v, want %v", test.delay, test.frequency, blockIndex, oscillator.value, want)
			}
		}
	}
}

func referenceLfoValue(blockIndex int, synth *Synthesizer, delay, frequency float32) float32 {
	currentTime := float64(blockIndex*int(synth.BlockSize)) / float64(synth.SampleRate)
	if currentTime < float64(delay) {
		return 0
	}
	period := 1 / float64(frequency)
	phase := math.Mod(currentTime-float64(delay), period) / period
	switch {
	case phase < 0.25:
		return float32(4 * phase)
	case phase < 0.75:
		return float32(4 * (0.5 - phase))
	default:
		return float32(4 * (phase - 1))
	}
}
