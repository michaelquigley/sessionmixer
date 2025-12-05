# sessionmixer - Development Notes

## Overview

`sessionmixer` is a custom mixer control surface application for Focusrite Scarlett audio interfaces. It uses the `dfx` immediate-mode GUI framework to provide a simplified, configurable interface for controlling hardware mixer settings via `scarlettctl`.

## Current Implementation Status

### Completed Features

1. **Core Architecture**
   - Configuration system (YAML-based)
   - Control mapping from config to hardware
   - Ganged fader support (mirror mode)
   - Event-driven hardware monitoring
   - Bidirectional update handling

2. **Configurable Fader Tapers**
   - `DecibelTaper(dB)` - logarithmic taper with configurable dB range (e.g., 72 for -60dB to +12dB)
   - `LinearTaper()` - linear taper (default if `taper_db` not specified)
   - Configured per-gang via `taper_db` field in config

3. **Level Metering**
   - Optional signal level display on fader track background
   - Logarithmic (dB) scale for sensitivity at low levels (96 dB range)
   - Color gradient: black (zero) -> dark green -> bright green -> yellow -> red (high)
   - Configured via `levels` field in gang config

4. **Bidirectional Update Strategy**
   - Implemented per docs/BIDIRECTIONAL_UPDATE_STRATEGY.md
   - Hardware as single source of truth
   - Value equality checks prevent feedback loops
   - Immediate writes (no debouncing)
   - Event-driven monitoring via scarlettctl.EventMonitor

### Files

- `config.go` - YAML configuration loading and validation
- `channel.go` - MixerChannel with bidirectional updates
- `gang.go` - GangedFader for controlling multiple channels with level metering
- `mapper.go` - Maps config to hardware controls
- `mixer.go` - Main GUI component (horizontal fader bank)
- `monitor.go` - Event monitoring for hardware changes
- `cmd/sessionmixer/` - Application entry point and commands

### Architecture

#### Data Flow

**UI -> Hardware:**
```
User drags fader
  |
dfx.FaderI detects change
  |
gang.HandleUIChange(newValue)
  |
Value equality check (prevents redundant writes)
  |
channel.HandleUIChange(value) for each ganged channel
  |
scarlettctl.Control.SetValue(value)
  |
ALSA write to hardware
```

**Hardware -> UI:**
```
Hardware state changes (external control)
  |
ALSA generates event
  |
EventMonitor.handleControlChange()
  |
gang.HandleHWChange(numID, newValue)
  |
channel.HandleHWChange(newValue)
  |
Value equality check (breaks feedback loop)
  |
Atomic update of cached value
  |
Next Draw() uses new value
```

**Level Metering (read-only):**
```
Every frame during Draw()
  |
gang.HasLevels() -> true
  |
gang.GetLevelColor()
  |
Read all level controls, find max
  |
Convert to dB scale (96 dB range)
  |
Map to HSV color gradient
  |
Set params.TrackColor
```

#### Key Components

**MixerChannel** (`channel.go`)
- Wraps a single hardware control
- Maintains atomic cached values (lastUIValue, lastHWValue)
- Implements value equality checks

**GangedFader** (`gang.go`)
- Controls multiple MixerChannels as a single fader
- Mirror mode: all channels get same value
- Configurable taper (DecibelTaper or LinearTaper)
- Optional level controls for signal visualization
- Computes track color from level meters using dB scale

**EventMonitor** (`monitor.go`)
- Runs scarlettctl.EventMonitor in goroutine
- Maps hardware events to gangs
- Thread-safe via atomic operations

**SessionMixer** (`mixer.go`)
- Main dfx.Component
- Horizontal scrollable fader bank
- Fixed-width table layout for stability

### Configuration

**Location:** `~/.config/sessionmixer/session.yaml`

**Structure:**
```yaml
card: 1  # ALSA card number

gang_controls:
  - name: "Mains"
    controls:
      - "Analogue 1 Playback Volume"
    unit: "db"
    taper_db: 72  # DecibelTaper with 72dB range

  - name: "MainMix"
    controls:
      - "Mix A Input 01 Playback Volume"
      - "Mix B Input 02 Playback Volume"
    levels:
      - "pcm:0.0/Level Meter[15]"
      - "pcm:0.0/Level Meter[16]"
    unit: "db"
    taper_db: 72
```

**Fields:**
- `name` - Display name for the fader
- `controls` - List of ALSA control names to gang together
- `unit` - Display unit: `"db"` (logarithmic dB display) or `"raw"` (integer value)
- `taper_db` - If > 0, use DecibelTaper with specified dB range; otherwise LinearTaper
- `levels` - Optional list of read-only level meter controls for signal visualization

### Dependencies

- `github.com/michaelquigley/dfx` - Immediate mode GUI framework
- `github.com/michaelquigley/scarlettctl` - Focusrite control library
- `github.com/michaelquigley/df/dd` - Configuration loading
- `github.com/AllenDang/cimgui-go/imgui` - ImGui bindings (via dfx)

### Building

```bash
go build ./cmd/sessionmixer
```

### Running

```bash
./sessionmixer run
```

## Next Steps

### High Priority
1. **Expand level metering visualization options**
   - Separate meter bars alongside faders
   - Peak hold indicators
   - RMS vs peak display modes

### Medium Priority
2. **Add more control types**
   - Boolean switches (Phantom, Air, Pad)
   - Enumerated controls (routing)
3. **Visual improvements**
   - Color coding for different control types
   - Visual indication of ganged controls
4. **Configuration improvements**
   - Auto-discovery mode (scan hardware and generate config)
   - Regex pattern matching for control names
   - Preset configurations for different workflows

### Low Priority
5. **Additional gang modes**
   - Relative mode (maintain offsets)
   - Scaled mode (scale to each control's range)
6. **Persistence**
   - Save/restore mixer state
   - Recall snapshots
7. **MIDI mapping**
   - Control faders via MIDI controllers

## Testing Notes

- Tested with Scarlett 16i16 4th Gen and 18i20 4th Gen
- Bidirectional updates work correctly (no feedback loops)
- Event monitoring is responsive
- Hardware writes are immediate
- dB display values match alsa-scarlett-gui formula
- DecibelTaper provides natural fader feel matching alsa-scarlett-gui

## Developer Notes

### Configuring Tapers

Tapers are configured per-gang in the YAML config:

```yaml
gang_controls:
  - name: "Volume"
    controls: ["..."]
    taper_db: 72   # DecibelTaper(72) - good for mixer volumes

  - name: "Pan"
    controls: ["..."]
    # taper_db omitted - uses LinearTaper
```

### Level Meter Colors

The level meter color gradient in `gang.go` uses HSV color space:
- 0-50%: dark green to bright green (increase brightness)
- 50-80%: green to yellow (shift hue)
- 80-100%: yellow to red (shift hue, max brightness)

The 96 dB dynamic range provides good sensitivity at low signal levels.

## References

- [docs/BIDIRECTIONAL_UPDATE_STRATEGY.md](docs/BIDIRECTIONAL_UPDATE_STRATEGY.md) - Strategy doc from alsa-scarlett-gui
- [dfx documentation](https://github.com/michaelquigley/dfx)
- [scarlettctl documentation](https://github.com/michaelquigley/scarlettctl)
