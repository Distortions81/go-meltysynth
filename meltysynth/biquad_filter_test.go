package meltysynth

import "testing"

func TestBiQuadFilterBlockLocalStateMatchesSampleLoop(t *testing.T) {
	got := biQuadFilter{
		active: true,
		a0:     0.2,
		a1:     0.3,
		a2:     0.1,
		a3:     -0.15,
		a4:     0.05,
		x1:     0.4,
		x2:     -0.2,
		y1:     0.1,
		y2:     -0.3,
	}
	want := got
	blocks := [][]float32{
		{0.1, -0.5, 0.25, 0.75, -0.9, 0.4},
		{-0.3, 0.8, 0.6, -0.2, 0.05, -0.7},
	}

	for blockIndex, input := range blocks {
		gotBlock := append([]float32(nil), input...)
		wantBlock := append([]float32(nil), input...)
		got.process(gotBlock)
		processBiQuadReference(&want, wantBlock)

		for i := range gotBlock {
			if gotBlock[i] != wantBlock[i] {
				t.Fatalf("block %d sample %d: got %v, want %v", blockIndex, i, gotBlock[i], wantBlock[i])
			}
		}
		if got.x1 != want.x1 || got.x2 != want.x2 || got.y1 != want.y1 || got.y2 != want.y2 {
			t.Fatalf("block %d: state mismatch", blockIndex)
		}
	}
}

func processBiQuadReference(filter *biQuadFilter, block []float32) {
	for t := range block {
		input := block[t]
		output := filter.a0*input + filter.a1*filter.x1 + filter.a2*filter.x2 - filter.a3*filter.y1 - filter.a4*filter.y2
		filter.x2 = filter.x1
		filter.x1 = input
		filter.y2 = filter.y1
		filter.y1 = output
		block[t] = output
	}
}
