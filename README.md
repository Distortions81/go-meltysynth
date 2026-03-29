# Go-MeltySynth

Go-MeltySynth is a SoundFont synthesizer written in Go, ported from [MeltySynth for C#](https://github.com/sinshu/meltysynth).



## Features

* Suitable for both real-time and offline synthesis.
* Support for standard MIDI files.
* No dependencies other than the standard library.



## Demo

https://www.youtube.com/watch?v=HLta6pASIFg

[![Youtube video](https://img.youtube.com/vi/HLta6pASIFg/0.jpg)](https://www.youtube.com/watch?v=HLta6pASIFg)



## Installation

```
go get github.com/sinshu/go-meltysynth
```



## Performance

Benchmarks in `meltysynth/benchmark_e1m1_test.go` use Doom's `D_E1M1` MUS data from `../GD-DOOM/DOOM1.WAD` with `../GD-DOOM/soundfonts/SGM-HQ.sf2`.
The harness fixes setup overhead by loading the SoundFont and parsing the MUS stream once, then benchmarking only `Synthesizer.Reset()` plus playback work.

Measured on March 29, 2026 on an AMD Ryzen 9 7950X, using the median of 5 runs:

| Benchmark | This repo | Upstream (`05d3113`) | Improvement |
| --- | ---: | ---: | ---: |
| `BenchmarkE1M1FirstChunkSGMHQ` | `98.678 µs/op` | `101.142 µs/op` | `2.44%` |
| `BenchmarkE1M1FullRenderSGMHQ` | `821.116 ms/op` | `844.117 ms/op` | `2.72%` |

Reproduce with:

```
MELTYSYNTH_E1M1_WAD=../GD-DOOM/DOOM1.WAD \
MELTYSYNTH_E1M1_SF2=../GD-DOOM/soundfonts/SGM-HQ.sf2 \
go test ./meltysynth -run '^$' -bench 'BenchmarkE1M1(FirstChunk|FullRender)SGMHQ$' -benchmem -count=5
```



## Bug Fixes

This fork also includes correctness fixes beyond the performance work:

* Fixed oscillator loop wrapping for looping samples so playback re-enters the loop relative to `startLoop` and stays correct even when pitch steps jump beyond the loop end.
* Fixed invalid looping sample handling by falling back to non-looped playback when `endLoop <= startLoop`, avoiding broken wrap behavior on malformed regions.
* Fixed active voice processing order in `renderBlock()` by updating the voice collection before reading `activeVoiceCount`, so rendering uses the current live voice set instead of a stale count.



## Examples

An example code to synthesize a simple chord:

```go
// Load the SoundFont.
sf2, _ := os.Open("TimGM6mb.sf2")
soundFont, _ := meltysynth.NewSoundFont(sf2)
sf2.Close()

// Create the synthesizer.
settings := meltysynth.NewSynthesizerSettings(44100)
synthesizer, _ := meltysynth.NewSynthesizer(soundFont, settings)

// Play some notes (middle C, E, G).
synthesizer.NoteOn(0, 60, 100)
synthesizer.NoteOn(0, 64, 100)
synthesizer.NoteOn(0, 67, 100)

// The output buffer (3 seconds).
length := 3 * settings.SampleRate
left := make([]float32, length)
right := make([]float32, length)

// Render the waveform.
synthesizer.Render(left, right)
```

Another example code to synthesize a MIDI file:

```go
// Load the SoundFont.
sf2, _ := os.Open("TimGM6mb.sf2")
soundFont, _ := meltysynth.NewSoundFont(sf2)
sf2.Close()

// Create the synthesizer.
settings := meltysynth.NewSynthesizerSettings(44100)
synthesizer, _ := meltysynth.NewSynthesizer(soundFont, settings)

// Load the MIDI file.
mid, _ := os.Open("C:\\Windows\\Media\\flourish.mid")
midiFile, _ := meltysynth.NewMidiFile(mid)
mid.Close()

// Create the MIDI sequencer.
sequencer := meltysynth.NewMidiFileSequencer(synthesizer)
sequencer.Play(midiFile, true)

// The output buffer.
length := int(float64(settings.SampleRate) * float64(midiFile.GetLength()) / float64(time.Second))
left := make([]float32, length)
right := make([]float32, length)

// Render the waveform.
sequencer.Render(left, right)
```



## Todo

* __Wave synthesis__
    - [x] SoundFont reader
    - [x] Waveform generator
    - [x] Envelope generator
    - [x] Low-pass filter
    - [x] Vibrato LFO
    - [x] Modulation LFO
* __MIDI message processing__
    - [x] Note on/off
    - [x] Bank selection
    - [x] Modulation
    - [x] Volume control
    - [x] Pan
    - [x] Expression
    - [x] Hold pedal
    - [x] Program change
    - [x] Pitch bend
    - [x] Tuning
* __Effects__
    - [x] Reverb
    - [x] Chorus
* __Other things__
    - [x] Standard MIDI file support
    - [x] Performace optimization



## License

Go-MeltySynth is available under [the MIT license](LICENSE.txt).
