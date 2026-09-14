package meltysynth

import "testing"

func TestSoundFontPreparedRegionsAreReusedBySynthesizer(t *testing.T) {
	sample := &SampleHeader{SampleRate: 44100, OriginalPitch: 60}
	instrumentRegion := &InstrumentRegion{Sample: sample}
	instrument := &Instrument{Regions: []*InstrumentRegion{instrumentRegion}}
	presetRegion := &PresetRegion{Instrument: instrument}
	preset := &Preset{Regions: []*PresetRegion{presetRegion}}
	soundFont := &SoundFont{Presets: []*Preset{preset}}
	soundFont.preparePresets()

	if len(soundFont.preparedPresets) != 1 || len(soundFont.preparedPresets[0].regions) != 1 {
		t.Fatal("SoundFont regions were not prepared")
	}
	prepared := soundFont.preparedPresets[0]

	synth, err := NewSynthesizer(soundFont, NewSynthesizerSettings(44100))
	if err != nil {
		t.Fatal(err)
	}
	if synth.defaultPreset != prepared {
		t.Fatal("synthesizer rebuilt prepared SoundFont regions")
	}
}

func TestSynthesizerPreparesLiteralSoundFont(t *testing.T) {
	sample := &SampleHeader{SampleRate: 44100, OriginalPitch: 60}
	instrumentRegion := &InstrumentRegion{Sample: sample}
	instrument := &Instrument{Regions: []*InstrumentRegion{instrumentRegion}}
	preset := &Preset{Regions: []*PresetRegion{{Instrument: instrument}}}
	soundFont := &SoundFont{Presets: []*Preset{preset}}

	synth, err := NewSynthesizer(soundFont, NewSynthesizerSettings(44100))
	if err != nil {
		t.Fatal(err)
	}
	if synth.defaultPreset == nil || len(synth.defaultPreset.regions) != 1 {
		t.Fatal("literal SoundFont regions were not prepared")
	}
}

func TestPreparedPresetIndexesKeyAndVelocityRegions(t *testing.T) {
	first := testInstrumentRegion(10, 70, 20, 127)
	second := testInstrumentRegion(40, 50, 0, 30)
	third := testInstrumentRegion(0, 127, 0, 127)
	firstPresetRegion := testPresetRegion(0, 60, 0, 100, first, second)
	secondPresetRegion := testPresetRegion(50, 127, 90, 127, third)
	prepared := newPreparedPreset(&Preset{Regions: []*PresetRegion{firstPresetRegion, secondPresetRegion}})

	tests := []struct {
		key      int32
		velocity int32
		want     []*InstrumentRegion
	}{
		{45, 25, []*InstrumentRegion{first, second}},
		{55, 95, []*InstrumentRegion{first, third}},
		{5, 50, nil},
		{100, 100, []*InstrumentRegion{third}},
		{-1, 100, nil},
		{60, 128, nil},
	}

	for _, test := range tests {
		got := prepared.regionsFor(test.key, test.velocity)
		if len(got) != len(test.want) {
			t.Fatalf("key %d velocity %d: got %d regions, want %d", test.key, test.velocity, len(got), len(test.want))
		}
		for i := range got {
			if got[i].instrumentRegion != test.want[i] {
				t.Fatalf("key %d velocity %d region %d: order or identity mismatch", test.key, test.velocity, i)
			}
		}
	}

	firstCell := 45*midiVelocityCount + 25
	adjacentCell := 45*midiVelocityCount + 26
	if prepared.noteOnLookup[firstCell] != prepared.noteOnLookup[adjacentCell] {
		t.Fatal("identical region sets did not share a lookup group")
	}

	for key := int32(0); key < midiKeyCount; key++ {
		for velocity := int32(0); velocity < midiVelocityCount; velocity++ {
			got := prepared.regionsFor(key, velocity)
			want := regionsForReference(prepared, key, velocity)
			if len(got) != len(want) {
				t.Fatalf("exhaustive key %d velocity %d: got %d regions, want %d", key, velocity, len(got), len(want))
			}
			for i := range got {
				if got[i] != want[i] {
					t.Fatalf("exhaustive key %d velocity %d region %d: order or identity mismatch", key, velocity, i)
				}
			}
		}
	}
}

func regionsForReference(prepared *preparedPreset, key int32, velocity int32) []*voiceParameters {
	var result []*voiceParameters
	for i := range prepared.regions {
		presetRegion := &prepared.regions[i]
		if !presetRegion.region.contains(key, velocity) {
			continue
		}
		for j := range presetRegion.instrumentRegions {
			params := &presetRegion.instrumentRegions[j].parameters
			if params.instrumentRegion.contains(key, velocity) {
				result = append(result, params)
			}
		}
	}
	return result
}

func testPresetRegion(keyStart int32, keyEnd int32, velocityStart int32, velocityEnd int32, regions ...*InstrumentRegion) *PresetRegion {
	region := &PresetRegion{Instrument: &Instrument{Regions: regions}}
	region.gs[gen_KeyRange] = int16(keyStart | keyEnd<<8)
	region.gs[gen_VelocityRange] = int16(velocityStart | velocityEnd<<8)
	return region
}

func testInstrumentRegion(keyStart int32, keyEnd int32, velocityStart int32, velocityEnd int32) *InstrumentRegion {
	region := &InstrumentRegion{Sample: &SampleHeader{SampleRate: 44100, OriginalPitch: 60}}
	region.gs[gen_KeyRange] = int16(keyStart | keyEnd<<8)
	region.gs[gen_VelocityRange] = int16(velocityStart | velocityEnd<<8)
	region.gs[gen_KeyNumber] = -1
	region.gs[gen_Velocity] = -1
	region.gs[gen_OverridingRootKey] = -1
	region.gs[gen_ScaleTuning] = 100
	return region
}
