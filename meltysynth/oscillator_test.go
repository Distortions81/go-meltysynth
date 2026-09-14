package meltysynth

import "testing"

func TestOscillatorNoLoopMatchesSampleLoop(t *testing.T) {
	data := []int16{-12000, -7000, 2000, 9000, 14000, 4000, -3000, 0}
	steps := []int64{fracUnit / 3, fracUnit, fracUnit + fracUnit/2, 7 * fracUnit}
	for _, step := range steps {
		got := oscillator{data: data, sampleEnd: 6, position_fp: fracUnit + fracUnit/4}
		want := got
		gotBlock := make([]float32, 12)
		wantBlock := make([]float32, 12)

		gotAlive := got.fillBlock_NoLoop(gotBlock, step)
		wantAlive := fillBlockNoLoopReference(&want, wantBlock, step)
		assertOscillatorResult(t, gotBlock, wantBlock, got.position_fp, want.position_fp, gotAlive, wantAlive)
	}
}

func TestOscillatorContinuousSpansMatchSampleLoop(t *testing.T) {
	data := []int16{-12000, -7000, 2000, 9000, 14000, 4000, -3000, 0}
	steps := []int64{fracUnit / 3, fracUnit, fracUnit + fracUnit/2, 7 * fracUnit}
	for _, step := range steps {
		got := oscillator{data: data, startLoop: 1, endLoop: 6, position_fp: fracUnit + fracUnit/4}
		want := got
		gotBlock := make([]float32, 20)
		wantBlock := make([]float32, 20)

		gotAlive := got.fillBlock_Continuous(gotBlock, step)
		wantAlive := fillBlockContinuousReference(&want, wantBlock, step)
		assertOscillatorResult(t, gotBlock, wantBlock, got.position_fp, want.position_fp, gotAlive, wantAlive)
	}
}

func assertOscillatorResult(t *testing.T, got []float32, want []float32, gotPosition int64, wantPosition int64, gotAlive bool, wantAlive bool) {
	t.Helper()
	if gotAlive != wantAlive {
		t.Fatalf("alive: got %v, want %v", gotAlive, wantAlive)
	}
	if gotPosition != wantPosition {
		t.Fatalf("position: got %d, want %d", gotPosition, wantPosition)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("sample %d: got %v, want %v", i, got[i], want[i])
		}
	}
}

func fillBlockNoLoopReference(o *oscillator, block []float32, pitchRatio_fp int64) bool {
	for t := range block {
		index := int32(o.position_fp >> fracBits)
		if index >= o.sampleEnd {
			if t > 0 {
				for i := t; i < len(block); i++ {
					block[i] = 0
				}
				return true
			}
			return false
		}

		x1 := int64(o.data[index])
		x2 := int64(o.data[index+1])
		a_fp := o.position_fp & (fracUnit - 1)
		block[t] = fpToSample * float32((x1<<fracBits)+a_fp*(x2-x1))
		o.position_fp += pitchRatio_fp
	}
	return true
}

func fillBlockContinuousReference(o *oscillator, block []float32, pitchRatio_fp int64) bool {
	startLoop_fp := int64(o.startLoop) << fracBits
	endLoop_fp := int64(o.endLoop) << fracBits
	loopLength := int32(o.endLoop - o.startLoop)
	loopLength_fp := int64(loopLength) << fracBits
	if loopLength_fp <= 0 {
		return fillBlockNoLoopReference(o, block, pitchRatio_fp)
	}

	for t := range block {
		if o.position_fp >= endLoop_fp {
			o.position_fp = startLoop_fp + (o.position_fp-startLoop_fp)%loopLength_fp
		}
		index1 := int32(o.position_fp >> fracBits)
		index2 := index1 + 1
		if index2 >= o.endLoop {
			index2 -= loopLength
		}
		x1 := int64(o.data[index1])
		x2 := int64(o.data[index2])
		a_fp := o.position_fp & (fracUnit - 1)
		block[t] = fpToSample * float32((x1<<fracBits)+a_fp*(x2-x1))
		o.position_fp += pitchRatio_fp
	}
	return true
}
