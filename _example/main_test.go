package main

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestWritePCMInterleavedInt16_SilentInput(t *testing.T) {
	var buf bytes.Buffer

	left := []float32{0, 0}
	right := []float32{0, 0}

	if err := writePCMInterleavedInt16(left, right, &buf); err != nil {
		t.Fatal(err)
	}

	got := make([]int16, 4)
	if err := binary.Read(&buf, binary.LittleEndian, got); err != nil {
		t.Fatal(err)
	}

	for i, sample := range got {
		if sample != 0 {
			t.Fatalf("sample %d = %d, want 0", i, sample)
		}
	}
}
