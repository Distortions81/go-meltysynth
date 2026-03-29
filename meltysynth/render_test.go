package meltysynth

import "testing"

func TestSynthesizerRender_UsesShorterStereoBuffer(t *testing.T) {
	s := &Synthesizer{
		BlockSize:    2,
		blockLeft:    []float32{1, 2},
		blockRight:   []float32{3, 4},
		blockRead:    0,
		MasterVolume: 1,
	}

	left := make([]float32, 2)
	right := make([]float32, 1)

	s.Render(left, right)

	if left[0] != 1 || left[1] != 0 {
		t.Fatalf("unexpected left buffer: %#v", left)
	}
	if right[0] != 3 {
		t.Fatalf("unexpected right buffer: %#v", right)
	}
}

func TestMidiFileSequencerRender_UsesShorterStereoBuffer(t *testing.T) {
	s := &Synthesizer{
		BlockSize:  2,
		blockLeft:  []float32{1, 2},
		blockRight: []float32{3, 4},
		blockRead:  0,
	}
	seq := &MidiFileSequencer{
		synthesizer: s,
		blockWrote:  0,
	}

	left := make([]float32, 2)
	right := make([]float32, 1)

	seq.Render(left, right)

	if left[0] != 1 || left[1] != 0 {
		t.Fatalf("unexpected left buffer: %#v", left)
	}
	if right[0] != 3 {
		t.Fatalf("unexpected right buffer: %#v", right)
	}
}
