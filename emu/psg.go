package emu

// PSG emulates the SN76489 Programmable Sound Generator
// - 3 square wave tone channels
// - 1 noise channel
// - 4-bit volume per channel (0 = max, 15 = silent)
type PSG struct {
	// Tone channel registers (10-bit frequency dividers)
	toneReg [3]uint16
	// Tone channel counters
	toneCounter [3]uint16
	// Tone channel output state (high/low)
	toneOutput [3]bool

	// Noise channel
	noiseReg     uint8  // 3-bit: NF1 NF0 FB (shift rate and feedback mode)
	noiseCounter uint16 // Counter for noise
	noiseShift   uint16 // 15-bit LFSR
	noiseOutput  bool

	// Volume registers (4-bit, 0=max, 15=off)
	volume [4]uint8 // 0-2 = tone channels, 3 = noise

	// Latch state for two-byte writes
	latchedChannel uint8 // Which channel is latched (0-3)
	latchedType    uint8 // 0 = tone/noise, 1 = volume

	// Clock info
	clocksPerSample float64
	clockCounter    float64
	clockDivider    int // Divides input clock by 16

	// Output buffer (used by GenerateSamples)
	buffer    []float32
	bufferPos int
}

// Volume table: converts 4-bit volume to linear amplitude
// 0 = maximum volume, 15 = silence
// Each step is approximately -2dB
var volumeTable = []float32{
	1.0, 0.794, 0.631, 0.501, 0.398, 0.316, 0.251, 0.200,
	0.158, 0.126, 0.100, 0.079, 0.063, 0.050, 0.040, 0.0,
}

// NewPSG creates a new PSG instance
// psgClock is the PSG clock frequency (typically 3579545 Hz for SMS)
// sampleRate is the audio output sample rate (e.g., 44100 Hz)
// bufferSize is the number of samples per buffer
func NewPSG(psgClock int, sampleRate int, bufferSize int) *PSG {
	p := &PSG{
		clocksPerSample: float64(psgClock) / float64(sampleRate),
		buffer:          make([]float32, bufferSize),
		noiseShift:      0x8000, // Initial LFSR state
	}
	// Initialize volumes to silent
	for i := range p.volume {
		p.volume[i] = 0x0F
	}
	return p
}

// Write handles writes to the PSG
func (p *PSG) Write(value uint8) {
	if value&0x80 != 0 {
		// LATCH/DATA byte: 1 CC T DDDD
		// CC = channel (0-2 tone, 3 noise)
		// T = type (0 = tone/noise, 1 = volume)
		// DDDD = data
		p.latchedChannel = (value >> 5) & 0x03
		p.latchedType = (value >> 4) & 0x01
		data := value & 0x0F

		if p.latchedType == 1 {
			// Volume write
			p.volume[p.latchedChannel] = data
		} else {
			// Tone/noise write
			if p.latchedChannel < 3 {
				// Tone channel: update low 4 bits
				p.toneReg[p.latchedChannel] = (p.toneReg[p.latchedChannel] & 0x3F0) | uint16(data)
			} else {
				// Noise channel control
				p.noiseReg = data & 0x07
				p.noiseShift = 0x8000 // Reset LFSR on noise reg write
			}
		}
	} else {
		// DATA byte: 0 X DDDDDD
		// Only valid for tone channels (not volume or noise)
		if p.latchedType == 0 && p.latchedChannel < 3 {
			// Tone channel: update high 6 bits
			data := uint16(value & 0x3F)
			p.toneReg[p.latchedChannel] = (p.toneReg[p.latchedChannel] & 0x0F) | (data << 4)
		}
	}
}

// Clock advances the PSG by one clock cycle (internal, doesn't generate samples)
func (p *PSG) Clock() {
	// SN76489 divides input clock by 16
	p.clockDivider++
	if p.clockDivider < 16 {
		return
	}
	p.clockDivider = 0

	// Update tone channels
	for i := 0; i < 3; i++ {
		if p.toneCounter[i] > 0 {
			p.toneCounter[i]--
		} else {
			// Reload counter and flip output
			if p.toneReg[i] == 0 {
				p.toneCounter[i] = 1
			} else {
				p.toneCounter[i] = p.toneReg[i]
			}
			p.toneOutput[i] = !p.toneOutput[i]
		}
	}

	// Update noise channel
	if p.noiseCounter > 0 {
		p.noiseCounter--
	} else {
		// Reload counter
		rate := p.noiseReg & 0x03
		switch rate {
		case 0:
			p.noiseCounter = 0x10
		case 1:
			p.noiseCounter = 0x20
		case 2:
			p.noiseCounter = 0x40
		case 3:
			// Use tone channel 2's frequency
			if p.toneReg[2] == 0 {
				p.noiseCounter = 1
			} else {
				p.noiseCounter = p.toneReg[2]
			}
		}

		// Shift LFSR and generate output
		p.noiseOutput = (p.noiseShift & 1) != 0

		// Save output bit before shift for feedback calculation
		outputBit := p.noiseShift & 1

		// Calculate feedback bit
		var feedback uint16
		if p.noiseReg&0x04 != 0 {
			// White noise: XOR bits 0 and 3 (pseudo-random sequence)
			feedback = ((p.noiseShift & 1) ^ ((p.noiseShift >> 3) & 1)) << 14
		} else {
			// Periodic noise: feedback the output bit only (repeating pattern)
			feedback = outputBit << 14
		}

		p.noiseShift = (p.noiseShift >> 1) | feedback
	}
}

// Sample generates one audio sample
func (p *PSG) Sample() float32 {
	var sample float32 = 0

	// Mix tone channels
	for i := 0; i < 3; i++ {
		if p.toneOutput[i] {
			sample += volumeTable[p.volume[i]]
		} else {
			sample -= volumeTable[p.volume[i]]
		}
	}

	// Mix noise channel
	if p.noiseOutput {
		sample += volumeTable[p.volume[3]]
	} else {
		sample -= volumeTable[p.volume[3]]
	}

	// Normalize (4 channels, each ±1 max)
	return sample / 4.0
}

// GenerateSamples fills the buffer with audio samples
// Called once per frame with the number of PSG clocks that occurred
func (p *PSG) GenerateSamples(clocks int) {
	p.bufferPos = 0

	for i := 0; i < clocks; i++ {
		p.Clock()
		p.clockCounter++

		// Generate sample when enough clocks have passed
		if p.clockCounter >= p.clocksPerSample {
			p.clockCounter -= p.clocksPerSample
			if p.bufferPos < len(p.buffer) {
				p.buffer[p.bufferPos] = p.Sample()
				p.bufferPos++
			}
		}
	}
}

// GetBuffer returns the current audio buffer and the number of valid samples
func (p *PSG) GetBuffer() ([]float32, int) {
	return p.buffer, p.bufferPos
}

// GetToneReg returns the 10-bit tone register for the given channel (0-2)
func (p *PSG) GetToneReg(ch int) uint16 {
	return p.toneReg[ch]
}

// GetVolume returns the 4-bit volume for the given channel (0-3)
func (p *PSG) GetVolume(ch int) uint8 {
	return p.volume[ch]
}

// GetNoiseReg returns the noise control register
func (p *PSG) GetNoiseReg() uint8 {
	return p.noiseReg
}

// GetVolumeTable returns the volume lookup table (for testing)
func GetVolumeTable() []float32 {
	return volumeTable
}
