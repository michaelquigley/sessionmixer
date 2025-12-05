# sessionmixer

A lightweight, configurable desktop mixer for controlling cue mixes and audio routing on audio interfaces with onboard mixing.

## Demo

<video src="docs/capture.webm" controls width="600"></video>

*sessionmixer controlling a Focusrite Scarlett interface with real-time level metering*

## Overview

**sessionmixer** provides an ergonomic control surface for audio interfaces that include onboard DSP mixing capabilities. Instead of navigating complex manufacturer software, sessionmixer lets you create a streamlined, personalized mixer layout with just the controls you need.

Whether you're managing monitor mixes in a recording session, routing audio for a podcast, or controlling a live streaming setup, sessionmixer gives you quick access to the controls that matter most.

### Supported Hardware

- **Focusrite Scarlett** (4th generation) - Tested with 16i16 and 18i20
- Additional interface support planned for future releases

## Features

- **Ganged Faders** - Control multiple hardware channels with a single fader (e.g., stereo pairs)
- **Configurable Tapers** - Choose between logarithmic (dB) or linear fader response
- **Level Metering** - Real-time signal levels displayed as color-coded fader backgrounds
- **Bidirectional Sync** - Changes made externally (other software, hardware controls) are reflected in the UI
- **YAML Configuration** - Simple, human-readable configuration files

## Requirements

- **Linux** with ALSA
- **Go 1.21+** (for building from source)
- A supported audio interface

### Dependencies

sessionmixer uses [scarlettctl](https://github.com/michaelquigley/scarlettctl) to communicate with Focusrite interfaces and [dfx](https://github.com/michaelquigley/dfx) for the user interface.

## Installation

```bash
# Clone the repository
git clone https://github.com/michaelquigley/sessionmixer.git
cd sessionmixer

# Build
go build ./cmd/sessionmixer

# Install (optional)
sudo cp sessionmixer /usr/local/bin/
```

## Configuration

sessionmixer uses a YAML configuration file located at:

```
~/.config/sessionmixer/session.yaml
```

### Example Configuration

```yaml
# ALSA card number (use `aplay -l` to find your device)
card: 1

# Define your faders
gang_controls:
  # A single-channel fader
  - name: "Mains"
    controls:
      - "Analogue 1 Playback Volume"
    unit: "db"
    taper_db: 72

  # A stereo gang with level metering
  - name: "Headphones"
    controls:
      - "Mix A Input 01 Playback Volume"
      - "Mix A Input 02 Playback Volume"
    levels:
      - "pcm:0.0/Level Meter[15]"
      - "pcm:0.0/Level Meter[16]"
    unit: "db"
    taper_db: 72
```

### Configuration Reference

| Field | Description |
|-------|-------------|
| `card` | ALSA card number for your interface |
| `gang_controls` | List of fader definitions |
| `name` | Display label for the fader |
| `controls` | ALSA control names to gang together |
| `unit` | Display format: `"db"` or `"raw"` |
| `taper_db` | dB range for logarithmic taper (omit for linear) |
| `levels` | Optional: level meter controls for signal display |

### Finding Control Names

Use `scarlettctl` to discover available controls on your interface:

```bash
scarlettctl list
```

## Usage

```bash
# Run the mixer
./sessionmixer run

# With verbose logging
./sessionmixer run -v
```

### Controls

- **Drag faders** to adjust levels
- Fader values sync bidirectionally with hardware
- Level meters (when configured) show real-time signal levels:
  - Green = normal levels
  - Yellow = approaching peak
  - Red = high levels

## License

MIT
