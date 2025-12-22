# sessionmixer

A lightweight, configurable desktop mixer for controlling cue mixes and audio routing on audio interfaces with onboard mixing.

## Project Roadmap

Take a look at the [project roadmap board](https://github.com/users/michaelquigley/projects/3/views/1) to better understand what's on deck for sessionmixer.

## Demo

![sessionmixer controlling a Focusrite Scarlett interface with real-time level metering](docs/screenshot.png)

*sessionmixer controlling a Focusrite Scarlett interface with real-time level metering*

## Overview

**sessionmixer** provides an ergonomic control surface for audio interfaces that include onboard DSP mixing capabilities. Instead of navigating complex manufacturer software, sessionmixer lets you create a streamlined, personalized mixer layout with just the controls you need.

Whether you're managing monitor mixes in a recording session, routing audio for a podcast, or controlling a live streaming setup, sessionmixer gives you quick access to the controls that matter most.

### Supported Hardware

- **Focusrite Scarlett** (4th generation) - Tested with 16i16 and 18i20
- Additional interface support planned for future releases

## Features

- **Automatic Device Detection** - Detects your Scarlett interface and loads the appropriate device profile
- **Device Topology Discovery** - Inspect your interface's ports, mixes, routing, and level meters
- **Ganged Faders** - Control multiple hardware channels with a single fader (e.g., stereo pairs)
- **Configurable Tapers** - Choose between logarithmic (dB) or linear fader response
- **Level Metering** - Real-time signal visualization with two modes:
  - **VU Meters** - Vertical meter bars alongside faders
  - **Track Color** - Fader background changes color based on signal level
- **Level Smoothing** - Configurable ring buffer averaging for smooth meter response
- **Bidirectional Sync** - Changes made externally (other software, hardware controls) are reflected in the UI
- **YAML Configuration** - Simple, human-readable configuration files

## Requirements

- **Linux** with ALSA
- **Go 1.24+** (for building from source)
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

# VU meter smoothing (number of samples to average, 0 = disabled)
level_smoothing: 8

# Define your faders
gang_controls:
  # A single-channel fader
  - name: "Mains"
    controls:
      - "Analogue 1 Playback Volume"
    unit: "db"
    taper_db: 72

  # A stereo gang with VU meters and track coloring
  - name: "Headphones"
    controls:
      - "Mix A Input 01 Playback Volume"
      - "Mix A Input 02 Playback Volume"
    levels:
      - "pcm:0.0/Level Meter[15]"
      - "pcm:0.0/Level Meter[16]"
    show_vu_meter: true
    show_track_color: true
    unit: "db"
    taper_db: 72
```

### Configuration Reference

**Root Fields:**

| Field | Description |
|-------|-------------|
| `card` | ALSA card number for your interface |
| `level_smoothing` | Number of samples to average for level meters (0 = disabled) |
| `gang_controls` | List of fader definitions |

**Gang Control Fields:**

| Field | Description |
|-------|-------------|
| `name` | Display label for the fader |
| `controls` | ALSA control names to gang together |
| `unit` | Display format: `"db"` or `"raw"` |
| `taper_db` | dB range for logarithmic taper (omit for linear) |
| `levels` | Level meter control names for signal visualization |
| `show_vu_meter` | Enable VU meter bars (requires `levels`) |
| `show_track_color` | Enable track coloring based on level (requires `levels`) |

### Finding Control Names

Use `scarlettctl` to discover available controls on your interface:

```bash
scarlettctl list
```

## Usage

```bash
# Run the mixer
./sessionmixer run

# Inspect device topology
./sessionmixer topology

# Show routing endpoints
./sessionmixer topology --routing

# Show mixes
./sessionmixer topology --mixes

# Show level meters
./sessionmixer topology --meters
```

### Controls

- **Drag faders** to adjust levels
- Fader values sync bidirectionally with hardware
- Level visualization (when `levels` are configured):
  - **VU Meters** - Vertical bar meters beside faders showing per-channel levels
  - **Track Color** - Fader background color indicates signal level
  - Color gradient: Green (normal) → Yellow (approaching peak) → Red (high levels)

## License

MIT
