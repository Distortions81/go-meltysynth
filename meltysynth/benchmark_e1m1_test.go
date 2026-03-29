package meltysynth

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

const (
	benchE1M1WADEnv = "MELTYSYNTH_E1M1_WAD"
	benchE1M1SF2Env = "MELTYSYNTH_E1M1_SF2"

	benchSampleRate = 44100
	benchTicRate    = 140
	benchChunk1024  = 1024
)

type benchMUSEvent struct {
	delta   uint32
	channel int32
	command int32
	data1   int32
	data2   int32
}

var (
	benchE1M1Once      sync.Once
	benchE1M1Events    []benchMUSEvent
	benchE1M1SoundFont *SoundFont
	benchE1M1Err       error
)

func BenchmarkE1M1FirstChunkSGMHQ(b *testing.B) {
	synth := benchmarkE1M1Synth(b)
	left := make([]float32, benchChunk1024)
	right := make([]float32, benchChunk1024)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		synth.Reset()
		benchmarkRenderFirstChunk(synth, benchmarkE1M1Events(b), left, right, benchChunk1024)
	}
}

func BenchmarkE1M1FullRenderSGMHQ(b *testing.B) {
	synth := benchmarkE1M1Synth(b)
	left := make([]float32, benchChunk1024)
	right := make([]float32, benchChunk1024)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		synth.Reset()
		benchmarkRenderFullSong(synth, benchmarkE1M1Events(b), left, right)
	}
}

func benchmarkE1M1Synth(b *testing.B) *Synthesizer {
	b.Helper()

	benchmarkLoadE1M1Assets()
	if benchE1M1Err != nil {
		b.Fatal(benchE1M1Err)
	}

	settings := NewSynthesizerSettings(benchSampleRate)
	synth, err := NewSynthesizer(benchE1M1SoundFont, settings)
	if err != nil {
		b.Fatalf("NewSynthesizer() error: %v", err)
	}
	return synth
}

func benchmarkE1M1Events(b *testing.B) []benchMUSEvent {
	b.Helper()

	benchmarkLoadE1M1Assets()
	if benchE1M1Err != nil {
		b.Fatal(benchE1M1Err)
	}
	return benchE1M1Events
}

func benchmarkLoadE1M1Assets() {
	benchE1M1Once.Do(func() {
		sf2Path := os.Getenv(benchE1M1SF2Env)
		if sf2Path == "" {
			sf2Path = filepath.Clean(filepath.Join("..", "GD-DOOM", "soundfonts", "SGM-HQ.sf2"))
		}

		sf2, err := os.Open(sf2Path)
		if err != nil {
			benchE1M1Err = fmt.Errorf("open soundfont %s: %w", sf2Path, err)
			return
		}
		benchE1M1SoundFont, err = NewSoundFont(sf2)
		sf2.Close()
		if err != nil {
			benchE1M1Err = fmt.Errorf("parse soundfont %s: %w", sf2Path, err)
			return
		}

		wadPath := os.Getenv(benchE1M1WADEnv)
		if wadPath == "" {
			for _, candidate := range []string{
				filepath.Clean(filepath.Join("..", "GD-DOOM", "DOOM.WAD")),
				filepath.Clean(filepath.Join("..", "GD-DOOM", "doom.wad")),
				filepath.Clean(filepath.Join("..", "GD-DOOM", "DOOM1.WAD")),
				filepath.Clean(filepath.Join("..", "GD-DOOM", "doom1.wad")),
			} {
				if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
					wadPath = candidate
					break
				}
			}
		}
		if wadPath == "" {
			benchE1M1Err = fmt.Errorf("set %s to a Doom IWAD containing D_E1M1", benchE1M1WADEnv)
			return
		}

		musData, err := benchmarkReadWADLump(wadPath, "D_E1M1")
		if err != nil {
			benchE1M1Err = err
			return
		}
		benchE1M1Events, err = benchmarkParseMUS(musData)
		if err != nil {
			benchE1M1Err = err
			return
		}
	})
}

func benchmarkRenderFirstChunk(synth *Synthesizer, events []benchMUSEvent, left []float32, right []float32, frames int) {
	remaining := frames
	for _, ev := range events {
		if ev.delta > 0 {
			waitFrames := benchmarkDeltaFrames(ev.delta)
			if waitFrames >= remaining {
				benchmarkRenderFrames(synth, remaining, left, right)
				return
			}
			benchmarkRenderFrames(synth, waitFrames, left, right)
			remaining -= waitFrames
		}
		synth.ProcessMidiMessage(ev.channel, ev.command, ev.data1, ev.data2)
		if remaining == 0 {
			return
		}
	}
	if remaining > 0 {
		benchmarkRenderFrames(synth, remaining, left, right)
	}
}

func benchmarkRenderFullSong(synth *Synthesizer, events []benchMUSEvent, left []float32, right []float32) {
	for _, ev := range events {
		if ev.delta > 0 {
			benchmarkRenderFrames(synth, benchmarkDeltaFrames(ev.delta), left, right)
		}
		synth.ProcessMidiMessage(ev.channel, ev.command, ev.data1, ev.data2)
	}
}

func benchmarkRenderFrames(synth *Synthesizer, frames int, left []float32, right []float32) {
	for frames > 0 {
		n := frames
		if n > len(left) {
			n = len(left)
		}
		synth.Render(left[:n], right[:n])
		frames -= n
	}
}

func benchmarkDeltaFrames(delta uint32) int {
	return int((uint64(delta) * uint64(benchSampleRate)) / uint64(benchTicRate))
}

func benchmarkReadWADLump(path string, lumpName string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read wad %s: %w", path, err)
	}
	if len(data) < 12 {
		return nil, fmt.Errorf("wad %s too short", path)
	}

	lumpCount := int(int32(binary.LittleEndian.Uint32(data[4:8])))
	dirOffset := int(int32(binary.LittleEndian.Uint32(data[8:12])))
	if lumpCount < 0 || dirOffset < 0 || dirOffset > len(data) {
		return nil, fmt.Errorf("wad %s has invalid directory", path)
	}
	if dirOffset+lumpCount*16 > len(data) {
		return nil, fmt.Errorf("wad %s directory truncated", path)
	}

	want := strings.ToUpper(lumpName)
	for i := 0; i < lumpCount; i++ {
		entry := dirOffset + i*16
		offset := int(int32(binary.LittleEndian.Uint32(data[entry : entry+4])))
		size := int(int32(binary.LittleEndian.Uint32(data[entry+4 : entry+8])))
		name := strings.TrimRight(string(data[entry+8:entry+16]), "\x00")
		if strings.ToUpper(name) != want {
			continue
		}
		if offset < 0 || size < 0 || offset+size > len(data) {
			return nil, fmt.Errorf("wad %s lump %s is out of bounds", path, lumpName)
		}
		return data[offset : offset+size], nil
	}

	return nil, fmt.Errorf("wad %s missing lump %s", path, lumpName)
}

func benchmarkParseMUS(data []byte) ([]benchMUSEvent, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("mus data too short: %d", len(data))
	}
	if string(data[:4]) != "MUS\x1a" {
		return nil, fmt.Errorf("bad mus signature %q", string(data[:4]))
	}

	scoreLen := int(binary.LittleEndian.Uint16(data[4:6]))
	scoreStart := int(binary.LittleEndian.Uint16(data[6:8]))
	if scoreStart < 0 || scoreStart >= len(data) {
		return nil, fmt.Errorf("invalid mus score start %d", scoreStart)
	}
	scoreEnd := scoreStart + scoreLen
	if scoreLen > 0 && scoreEnd <= len(data) {
		data = data[:scoreEnd]
	}

	pos := scoreStart
	velocity := [16]uint8{}
	for i := range velocity {
		velocity[i] = 127
	}
	var (
		channelMap   benchMUSChannelMap
		pendingDelta uint32
		events       = make([]benchMUSEvent, 0, scoreLen)
	)

	appendEvent := func(channel byte, command int32, data1 uint8, data2 uint8) {
		events = append(events, benchMUSEvent{
			delta:   pendingDelta,
			channel: int32(channel),
			command: command,
			data1:   int32(data1),
			data2:   int32(data2),
		})
		pendingDelta = 0
	}

	for pos < len(data) {
		evb := data[pos]
		pos++
		last := (evb & 0x80) != 0
		rawChannel := evb & 0x0F
		musType := (evb >> 4) & 0x07
		channel := channelMap.midiChannel(rawChannel)

		switch musType {
		case 0:
			if pos >= len(data) {
				return nil, fmt.Errorf("truncated mus note-off")
			}
			appendEvent(channel, 0x80, data[pos]&0x7F, 0)
			pos++
		case 1:
			if pos >= len(data) {
				return nil, fmt.Errorf("truncated mus note-on")
			}
			note := data[pos]
			pos++
			if (note & 0x80) != 0 {
				if pos >= len(data) {
					return nil, fmt.Errorf("truncated mus note-on velocity")
				}
				velocity[channel] = data[pos] & 0x7F
				pos++
			}
			appendEvent(channel, 0x90, note&0x7F, velocity[channel])
		case 2:
			if pos >= len(data) {
				return nil, fmt.Errorf("truncated mus pitch bend")
			}
			pitch := uint16(data[pos]) << 6
			pos++
			appendEvent(channel, 0xE0, uint8(pitch&0x7F), uint8((pitch>>7)&0x7F))
		case 3:
			if pos >= len(data) {
				return nil, fmt.Errorf("truncated mus system event")
			}
			if cc, ok := benchmarkMUSSystemToControl(data[pos]); ok {
				appendEvent(channel, 0xB0, cc, 0)
			}
			pos++
		case 4:
			if pos+1 >= len(data) {
				return nil, fmt.Errorf("truncated mus controller")
			}
			ctrl := data[pos]
			val := data[pos+1] & 0x7F
			pos += 2
			if ctrl == 0 {
				appendEvent(channel, 0xC0, val, 0)
				break
			}
			if cc, ok := benchmarkMUSControllerToMIDI(ctrl); ok {
				appendEvent(channel, 0xB0, cc, val)
			}
		case 5:
		case 6:
			return events, nil
		default:
			return nil, fmt.Errorf("unsupported mus event type %d", musType)
		}

		if last {
			delta, n, err := benchmarkReadMUSVarLen(data[pos:])
			if err != nil {
				return nil, err
			}
			pos += n
			pendingDelta = delta
		}
	}

	return events, nil
}

func benchmarkReadMUSVarLen(data []byte) (uint32, int, error) {
	var value uint32
	for i := 0; i < len(data); i++ {
		b := data[i]
		value = (value << 7) | uint32(b&0x7F)
		if (b & 0x80) == 0 {
			return value, i + 1, nil
		}
	}
	return 0, 0, fmt.Errorf("truncated mus delta-time")
}

func benchmarkMUSControllerToMIDI(ctrl uint8) (uint8, bool) {
	switch ctrl {
	case 1:
		return 32, true
	case 2:
		return 1, true
	case 3:
		return 7, true
	case 4:
		return 10, true
	case 5:
		return 11, true
	case 6:
		return 91, true
	case 7:
		return 93, true
	case 8:
		return 64, true
	case 9:
		return 67, true
	default:
		return 0, false
	}
}

func benchmarkMUSSystemToControl(ctrl uint8) (uint8, bool) {
	switch ctrl {
	case 10:
		return 120, true
	case 11:
		return 123, true
	case 12:
		return 126, true
	case 13:
		return 127, true
	case 14:
		return 121, true
	default:
		return 0, false
	}
}

type benchMUSChannelMap struct {
	next byte
	m    [16]byte
	set  [16]bool
}

func (m *benchMUSChannelMap) midiChannel(musChannel byte) byte {
	if musChannel == 15 {
		return 9
	}
	if m.set[musChannel] {
		return m.m[musChannel]
	}
	channel := m.next
	if channel == 9 {
		channel++
	}
	m.m[musChannel] = channel
	m.set[musChannel] = true
	m.next = channel + 1
	if m.next == 9 {
		m.next++
	}
	return channel
}
