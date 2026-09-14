package meltysynth

import "math"

type lfo struct {
	synthesizer    *Synthesizer
	active         bool
	phase          float64
	phaseIncrement float64
	value          float32
}

func (lfo *lfo) start(delay float32, frequency float32) {
	if frequency > 1.0e-3 {
		lfo.active = true

		lfo.phase = -float64(delay) * float64(frequency)
		lfo.phaseIncrement = float64(lfo.synthesizer.BlockSize) * float64(frequency) / float64(lfo.synthesizer.SampleRate)
		lfo.value = 0
		return
	}
	lfo.active = false
	lfo.value = 0
}

func (lfo *lfo) process() {
	if !lfo.active {
		return
	}

	lfo.phase += lfo.phaseIncrement
	if lfo.phase < 0 {
		lfo.value = 0
		return
	}
	if lfo.phase >= 1 {
		if lfo.phase < 2 {
			lfo.phase--
		} else {
			lfo.phase -= math.Floor(lfo.phase)
		}
	}

	switch {
	case lfo.phase < 0.25:
		lfo.value = float32(4 * lfo.phase)
	case lfo.phase < 0.75:
		lfo.value = float32(4 * (0.5 - lfo.phase))
	default:
		lfo.value = float32(4 * (lfo.phase - 1.0))
	}
}
