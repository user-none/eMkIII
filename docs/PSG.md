# Sega Master System PSG Integration Reference

This document covers how the SN76489-compatible PSG (Programmable Sound
Generator) is integrated into the Sega Master System at the system level. It
does not document the SN76489 chip internals (registers, write protocol,
tone/noise generation, volume attenuation, LFSR behavior, etc.). For chip-level
documentation, refer to the SN76489 datasheet or the go-chip-sn76489 module
which implements the chip emulation.

## Table of Contents

- [Physical Integration](#physical-integration)
- [I/O Port Mapping](#io-port-mapping)
- [Clock Source](#clock-source)
- [Audio Output](#audio-output)
- [Audio Timing and Frame Relationship](#audio-timing-and-frame-relationship)
- [Game Gear Stereo Extension](#game-gear-stereo-extension)
- [Sources](#sources)

---

## Physical Integration

The PSG is not a separate discrete chip on the SMS motherboard. It is
**physically integrated into the VDP die**. The VDP chip (Sega part number
315-5124, manufactured by Yamaha as the YM2602B) contains both the video
display processor and an SN76489-compatible PSG core on a single piece of
silicon. Audio output comes directly from pin 10 of the VDP IC.

The official SMS Service Manual lists every IC on the motherboard. There is no
SN76489 listed -- the VDP (IC5) is the sole source of audio.

This integration applies to all SMS-family VDP variants:

| Chip     | System                             |
|----------|-----------------------------------|
| 315-5124 | Mark III, Master System (SMS1)    |
| 315-5246 | Master System II (SMS2), late SMS1 |
| 315-5378 | Game Gear                         |

The integration predates the Genesis/Mega Drive. The original SG-1000 used a
discrete SN76489AN alongside a separate TMS9918 VDP. Starting with later
SG-1000 II revisions (315-5066), Yamaha combined the VDP and PSG onto a single
die. Every Sega console from the Mark III onward has used this integrated
approach.

Because the PSG core in the Yamaha VDPs is derived from the YM2149 rather than
being a direct copy of the TI silicon, there are minor behavioral differences
from a genuine TI SN76489 (LFSR taps, frequency-zero behavior, etc.). These
are Sega-variant chip details handled by the PSG emulation module, not
system-level integration concerns.

---

## I/O Port Mapping

The PSG is accessed via Z80 `OUT` instructions. It is **write-only** from the
CPU's perspective. There is no way to read back PSG state.

The SMS uses partial address decoding on bits A7, A6, and A0. The PSG responds
to writes in the port range `$40`-`$7F`:

| Port Range        | Read             | Write        |
|------------------|------------------|-------------|
| `$40`-`$7F` even | VDP V-counter   | PSG data    |
| `$40`-`$7F` odd  | VDP H-counter   | PSG data    |

The canonical PSG port is `$7F`, though any address in `$40`-`$7F` (even or
odd) works identically for writes. A few games write to `$7E` instead.

The asymmetric behavior of this port range is notable: **reads and writes to
the same addresses access completely different hardware**. Writes go to the PSG
inside the VDP. Reads return VDP scanline counters (V-counter from even
addresses, H-counter from odd addresses). The PSG data bus is 8-bit write-only.

---

## Clock Source

The PSG receives the Z80 CPU clock as its input clock. This clock is derived
from the system master clock (which is 3x the CPU clock):

| Region | CPU Clock (PSG Input) | Master Clock   |
|--------|----------------------|----------------|
| NTSC   | 3,579,545 Hz         | 10,738,635 Hz  |
| PAL    | 3,546,893 Hz         | 10,640,679 Hz  |

The PSG internally divides this input clock by 16 via a prescaler. The
prescaled clock (~223 kHz NTSC, ~222 kHz PAL) is what drives the tone counters
and noise LFSR.

Because the CPU and PSG share the same clock source, they are fully
synchronous. There is no clock domain crossing or drift between CPU execution
and PSG output.

---

## Audio Output

The SMS outputs **mono audio**. All four PSG channels (3 tone + 1 noise) are
mixed internally by the PSG into a single analog signal. This mono signal is
routed from the VDP's audio output pin through the console's A/V connector
(composite video or RGB).

There is no hardware mixer, amplifier, or audio DAC external to the VDP on the
SMS motherboard. The analog audio signal produced by the PSG core inside the
VDP is the final audio output.

---

## Audio Timing and Frame Relationship

There is no explicit hardware synchronization mechanism between PSG audio
output and VDP video rendering. The PSG runs continuously and independently of
the video rasterizer. However, because they share the same master clock, they
are phase-locked and do not drift relative to each other.

In practice, games update PSG registers during the **VBlank interrupt**,
naturally aligning audio updates with the frame boundary (~60 Hz NTSC, ~50 Hz
PAL). Some games that perform PCM sample playback through volume register
manipulation update the PSG much more frequently, consuming significant CPU
time.

For emulation, audio samples are generated per-scanline based on the number of
CPU cycles consumed during that scanline. Samples are accumulated across all
scanlines in a frame and then output as a batch. This approach maintains
correct timing alignment between audio and video without requiring
cycle-exact PSG stepping.

---

## Game Gear Stereo Extension

The Game Gear added a stereo panning register at **I/O port `$06`**
(write-only). This register does not exist on the SMS. On SMS hardware, port
`$06` is not connected to the PSG.

### Register Format

```
Bit 7: Noise channel  -> Left speaker
Bit 6: Tone channel 2 -> Left speaker
Bit 5: Tone channel 1 -> Left speaker
Bit 4: Tone channel 0 -> Left speaker
Bit 3: Noise channel  -> Right speaker
Bit 2: Tone channel 2 -> Right speaker
Bit 1: Tone channel 1 -> Right speaker
Bit 0: Tone channel 0 -> Right speaker
```

- Setting a bit to 1 enables that channel on that speaker
- Default value: `$FF` (all channels to both speakers = mono)
- `$00` mutes all output
- `$F0` routes all channels to left only
- `$0F` routes all channels to right only

### Speaker vs Headphone Behavior

The built-in Game Gear speaker is **not affected** by this register. It always
outputs all channels as mono regardless of the stereo panning setting. Stereo
separation only applies to the **headphone jack output**. This means sounds
panned to one side will be relatively louder through the speaker (which plays
everything) than through headphones (which play only the panned side).

---

## Sources

- Charles MacDonald, "Sega Master System VDP Documentation"
  (msvdp-20021112.txt) -- States "All versions of the VDP also have a Texas
  Instruments SN76489 sound chip built in."
  https://www.smspower.org/uploads/Development/msvdp-20021112.txt

- SMS Power!, "SN76489 Development Page"
  https://www.smspower.org/Development/SN76489

- SMS Power!, "I/O Port Map"
  https://www.smspower.org/Development/IOPortMap

- SMS Power!, "Audio Control Port" (Game Gear stereo register)
  https://www.smspower.org/Development/AudioControlPort

- Sega Master System Service Manual -- IC listing confirms no discrete
  SN76489 on the motherboard; audio output from VDP (IC5) pin 10.

- Rodrigo Copetti, "Master System Architecture" -- Confirms PSG is embedded
  in the VDP chip.
  https://www.copetti.org/writings/consoles/master-system/

- SMS Power!, "315-5124 Die Shot" -- Yamaha YM2602B die photo confirming
  integrated PSG.
  https://www.smspower.org/Development/315-5124Die

- SMS Power!, "Development Documents Index"
  https://www.smspower.org/Development/Documents

- franckverrot/EmulationResources, SMS/GG technical documents
  https://github.com/franckverrot/EmulationResources/tree/master/consoles/sms-gg
