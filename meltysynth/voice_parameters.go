package meltysynth

const (
	midiKeyCount      = 128
	midiVelocityCount = 128
	noteOnLookupSize  = midiKeyCount * midiVelocityCount
)

type envelopeParameters struct {
	delay            float32
	attack           float32
	hold             float32
	decay            float32
	sustain          float32
	release          float32
	keyNumberToHold  int32
	keyNumberToDecay int32
}

type oscillatorParameters struct {
	loopMode    int32
	sampleRate  int32
	sampleStart int32
	sampleEnd   int32
	startLoop   int32
	endLoop     int32
	rootKey     int32
	coarseTune  int32
	fineTune    int32
	scaleTuning int32
}

type voiceParameters struct {
	instrumentRegion  *InstrumentRegion
	exclusiveClass    int32
	sampleAttenuation float32
	filterAttenuation float32
	cutoff            float32
	resonance         float32

	vibLfoToPitch  float32
	modLfoToPitch  float32
	modEnvToPitch  float32
	modLfoToCutoff int32
	modEnvToCutoff int32
	modLfoToVolume float32
	dynamicCutoff  bool
	dynamicVolume  bool

	instrumentPan    float32
	instrumentReverb float32
	instrumentChorus float32

	volumeEnvelope         envelopeParameters
	modulationEnvelope     envelopeParameters
	vibratoLfoDelay        float32
	vibratoLfoFrequency    float32
	modulationLfoDelay     float32
	modulationLfoFrequency float32
	oscillator             oscillatorParameters
}

type preparedInstrumentRegion struct {
	parameters voiceParameters
}

type preparedPresetRegion struct {
	region            *PresetRegion
	instrumentRegions []preparedInstrumentRegion
}

type preparedPreset struct {
	preset       *Preset
	regions      []preparedPresetRegion
	noteOnLookup [noteOnLookupSize]uint16
	noteOnGroups [][]*voiceParameters
}

func newPreparedPreset(preset *Preset) *preparedPreset {
	result := &preparedPreset{
		preset:  preset,
		regions: make([]preparedPresetRegion, len(preset.Regions)),
	}
	for i, presetRegion := range preset.Regions {
		preparedRegion := &result.regions[i]
		preparedRegion.region = presetRegion
		preparedRegion.instrumentRegions = make([]preparedInstrumentRegion, len(presetRegion.Instrument.Regions))
		for j, instrumentRegion := range presetRegion.Instrument.Regions {
			pair := newRegionPair(presetRegion, instrumentRegion)
			preparedRegion.instrumentRegions[j] = preparedInstrumentRegion{
				parameters: newVoiceParameters(pair),
			}
		}
	}
	result.prepareNoteOnLookup()
	return result
}

func (p *preparedPreset) prepareNoteOnLookup() {
	cellRegions := make([][]int, noteOnLookupSize)
	parameters := make([]*voiceParameters, 0)

	for i := range p.regions {
		presetRegion := &p.regions[i]
		presetKeyStart := presetRegion.region.GetKeyRangeStart()
		presetKeyEnd := presetRegion.region.GetKeyRangeEnd()
		presetVelocityStart := presetRegion.region.GetVelocityRangeStart()
		presetVelocityEnd := presetRegion.region.GetVelocityRangeEnd()

		for j := range presetRegion.instrumentRegions {
			instrumentRegion := &presetRegion.instrumentRegions[j]
			region := instrumentRegion.parameters.instrumentRegion
			keyStart := maxInt32(presetKeyStart, region.GetKeyRangeStart())
			keyEnd := minInt32(presetKeyEnd, region.GetKeyRangeEnd())
			velocityStart := maxInt32(presetVelocityStart, region.GetVelocityRangeStart())
			velocityEnd := minInt32(presetVelocityEnd, region.GetVelocityRangeEnd())
			if keyStart > keyEnd || velocityStart > velocityEnd || keyEnd < 0 || keyStart >= midiKeyCount || velocityEnd < 0 || velocityStart >= midiVelocityCount {
				continue
			}

			keyStart = maxInt32(keyStart, 0)
			keyEnd = minInt32(keyEnd, midiKeyCount-1)
			velocityStart = maxInt32(velocityStart, 0)
			velocityEnd = minInt32(velocityEnd, midiVelocityCount-1)
			parameterIndex := len(parameters)
			parameters = append(parameters, &instrumentRegion.parameters)
			for key := keyStart; key <= keyEnd; key++ {
				row := int(key) * midiVelocityCount
				for velocity := velocityStart; velocity <= velocityEnd; velocity++ {
					cell := row + int(velocity)
					cellRegions[cell] = append(cellRegions[cell], parameterIndex)
				}
			}
		}
	}

	// Most key/velocity cells select the same region set. Store those sets once
	// and use a compact group ID for each cell.
	p.noteOnGroups = [][]*voiceParameters{nil}
	groupIndices := [][]int{nil}
	groupsByHash := map[uint64][]uint16{hashRegionIndices(nil): {0}}
	for cell, indices := range cellRegions {
		if len(indices) == 0 {
			continue
		}

		hash := hashRegionIndices(indices)
		groupID := uint16(0)
		for _, candidateID := range groupsByHash[hash] {
			if equalRegionIndices(indices, groupIndices[candidateID]) {
				groupID = candidateID
				break
			}
		}
		if groupID == 0 {
			groupID = uint16(len(p.noteOnGroups))
			storedIndices := append([]int(nil), indices...)
			group := make([]*voiceParameters, len(indices))
			for i, parameterIndex := range indices {
				group[i] = parameters[parameterIndex]
			}
			groupIndices = append(groupIndices, storedIndices)
			p.noteOnGroups = append(p.noteOnGroups, group)
			groupsByHash[hash] = append(groupsByHash[hash], groupID)
		}
		p.noteOnLookup[cell] = groupID
	}
}

func (p *preparedPreset) regionsFor(key int32, velocity int32) []*voiceParameters {
	if key < 0 || key >= midiKeyCount || velocity < 0 || velocity >= midiVelocityCount {
		return nil
	}
	cell := int(key)*midiVelocityCount + int(velocity)
	return p.noteOnGroups[p.noteOnLookup[cell]]
}

func hashRegionIndices(indices []int) uint64 {
	const (
		offset = uint64(14695981039346656037)
		prime  = uint64(1099511628211)
	)
	hash := offset
	for _, index := range indices {
		value := uint64(index + 1)
		for i := 0; i < 8; i++ {
			hash ^= value & 0xFF
			hash *= prime
			value >>= 8
		}
	}
	return hash
}

func equalRegionIndices(a []int, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func minInt32(a int32, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func maxInt32(a int32, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func newVoiceParameters(region regionPair) voiceParameters {
	filterQ := region.GetInitialFilterQ()
	modLfoToCutoff := region.GetModulationLfoToFilterCutoffFrequency()
	modEnvToCutoff := region.GetModulationEnvelopeToFilterCutoffFrequency()
	modLfoToVolume := region.GetModulationLfoToVolume()
	volumeRelease := region.GetReleaseVolumeEnvelope()
	if volumeRelease < 0.01 {
		volumeRelease = 0.01
	}

	return voiceParameters{
		instrumentRegion:  region.instrument,
		exclusiveClass:    region.GetExclusiveClass(),
		sampleAttenuation: 0.4 * region.GetInitialAttenuation(),
		filterAttenuation: 0.5 * filterQ,
		cutoff:            region.GetInitialFilterCutoffFrequency(),
		resonance:         calcDecibelsToLinear(filterQ),
		vibLfoToPitch:     0.01 * float32(region.GetVibratoLfoToPitch()),
		modLfoToPitch:     0.01 * float32(region.GetModulationLfoToPitch()),
		modEnvToPitch:     0.01 * float32(region.GetModulationEnvelopeToPitch()),
		modLfoToCutoff:    modLfoToCutoff,
		modEnvToCutoff:    modEnvToCutoff,
		modLfoToVolume:    modLfoToVolume,
		dynamicCutoff:     modLfoToCutoff != 0 || modEnvToCutoff != 0,
		dynamicVolume:     modLfoToVolume > 0.05,
		instrumentPan:     calcClamp(region.GetPan(), -50, 50),
		instrumentReverb:  0.01 * region.GetReverbEffectsSend(),
		instrumentChorus:  0.01 * region.GetChorusEffectsSend(),
		volumeEnvelope: envelopeParameters{
			delay:            region.GetDelayVolumeEnvelope(),
			attack:           region.GetAttackVolumeEnvelope(),
			hold:             region.GetHoldVolumeEnvelope(),
			decay:            region.GetDecayVolumeEnvelope(),
			sustain:          calcDecibelsToLinear(-region.GetSustainVolumeEnvelope()),
			release:          volumeRelease,
			keyNumberToHold:  region.GetKeyNumberToVolumeEnvelopeHold(),
			keyNumberToDecay: region.GetKeyNumberToVolumeEnvelopeDecay(),
		},
		modulationEnvelope: envelopeParameters{
			delay:            region.GetDelayModulationEnvelope(),
			attack:           region.GetAttackModulationEnvelope(),
			hold:             region.GetHoldModulationEnvelope(),
			decay:            region.GetDecayModulationEnvelope(),
			sustain:          1 - region.GetSustainModulationEnvelope()/100,
			release:          region.GetReleaseModulationEnvelope(),
			keyNumberToHold:  region.GetKeyNumberToModulationEnvelopeHold(),
			keyNumberToDecay: region.GetKeyNumberToModulationEnvelopeDecay(),
		},
		vibratoLfoDelay:        region.GetDelayVibratoLfo(),
		vibratoLfoFrequency:    region.GetFrequencyVibratoLfo(),
		modulationLfoDelay:     region.GetDelayModulationLfo(),
		modulationLfoFrequency: region.GetFrequencyModulationLfo(),
		oscillator: oscillatorParameters{
			loopMode:    region.GetSampleModes(),
			sampleRate:  region.instrument.Sample.SampleRate,
			sampleStart: region.GetSampleStart(),
			sampleEnd:   region.GetSampleEnd(),
			startLoop:   region.GetSampleStartLoop(),
			endLoop:     region.GetSampleEndLoop(),
			rootKey:     region.GetRootKey(),
			coarseTune:  region.GetCoarseTune(),
			fineTune:    region.GetFineTune(),
			scaleTuning: region.GetScaleTuning(),
		},
	}
}
