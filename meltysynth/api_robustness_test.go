package meltysynth

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

type shortReadReader struct {
	data      []byte
	chunkSize int
}

func (r *shortReadReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}

	n := len(p)
	if n > r.chunkSize {
		n = r.chunkSize
	}
	if n > len(r.data) {
		n = len(r.data)
	}

	copy(p[:n], r.data[:n])
	r.data = r.data[n:]
	return n, nil
}

func TestReadFourCC_AllowsShortReads(t *testing.T) {
	got, err := readFourCC(&shortReadReader{
		data:      []byte("RIFF"),
		chunkSize: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "RIFF" {
		t.Fatalf("got %q, want %q", got, "RIFF")
	}
}

func TestReadFixedLengthString_AllowsShortReads(t *testing.T) {
	got, err := readFixedLengthString(&shortReadReader{
		data:      []byte{'t', 'e', 's', 't', 0},
		chunkSize: 2,
	}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if got != "test" {
		t.Fatalf("got %q, want %q", got, "test")
	}
}

func TestNewSoundFont_NilReader(t *testing.T) {
	_, err := NewSoundFont(nil)
	if err == nil || err.Error() != "reader must not be nil" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewMidiFile_NilReader(t *testing.T) {
	_, err := NewMidiFile(nil)
	if err == nil || err.Error() != "reader must not be nil" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewSynthesizer_RejectsNilInputs(t *testing.T) {
	settings := NewSynthesizerSettings(44100)

	_, err := NewSynthesizer(nil, settings)
	if err == nil || err.Error() != "soundfont must not be nil" {
		t.Fatalf("unexpected soundfont error: %v", err)
	}

	sf := &SoundFont{}
	_, err = NewSynthesizer(sf, nil)
	if err == nil || err.Error() != "settings must not be nil" {
		t.Fatalf("unexpected settings error: %v", err)
	}
}

func TestMidiFileGetLength_Empty(t *testing.T) {
	var mf MidiFile
	if got := mf.GetLength(); got != 0 {
		t.Fatalf("got %v, want 0", got)
	}
}

func TestNewMidiFileSequencer_NilSynthesizer(t *testing.T) {
	_, err := NewMidiFileSequencer(nil)
	if err == nil || err.Error() != "synthesizer must not be nil" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReadTempo_AllowsShortReads(t *testing.T) {
	reader := &shortReadReader{
		data:      []byte{0x03, 0x07, 0xA1, 0x20},
		chunkSize: 1,
	}
	got, err := readTempo(reader)
	if err != nil {
		t.Fatal(err)
	}
	if got != 500000 {
		t.Fatalf("got %d, want %d", got, 500000)
	}
}

func TestDiscardData_AllowsShortReads(t *testing.T) {
	reader := &shortReadReader{
		data:      []byte{0x03, 0x01, 0x02, 0x03},
		chunkSize: 1,
	}
	if err := discardData(reader); err != nil {
		t.Fatal(err)
	}
}

func TestReadHelpers_PropagateEOF(t *testing.T) {
	_, err := readFourCC(bytes.NewReader([]byte("RI")))
	if !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		t.Fatalf("unexpected error: %v", err)
	}
}
