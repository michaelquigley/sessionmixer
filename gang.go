package sessionmixer

import (
	"fmt"
	"log"
	"math"
	"sync/atomic"

	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/michaelquigley/dfx"
	"github.com/michaelquigley/scarlettctl"
)

// GangMode specifies how ganged controls are synchronized
type GangMode string

const (
	// GangModeMirror - all controls get the same value
	GangModeMirror GangMode = "mirror"

	// GangModeRelative - maintains relative offsets between controls (future)
	GangModeRelative GangMode = "relative"

	// GangModeScaled - scales to each control's range (future)
	GangModeScaled GangMode = "scaled"
)

// GangedFader represents a single fader that controls multiple hardware controls
// This implements "ganging" where one UI fader controls multiple channels
type GangedFader struct {
	// Display properties
	name string
	unit string
	mode GangMode

	// The channels being ganged together
	channels []*MixerChannel

	// Cached value (represents the fader position)
	// For mirror mode, this is the common value
	// For relative/scaled modes, this would be the normalized position
	lastValue int64

	// dfx fader parameters
	params dfx.FaderParams

	// Min/max for the fader (derived from first channel in mirror mode)
	min int64
	max int64

	// Taper configuration
	taperDb float32 // If > 0, use DecibelTaper; otherwise LinearTaper

	// Level controls for signal indication (read-only)
	levelControls []*scarlettctl.Control
	levelMin      int64
	levelMax      int64
}

// NewGangedFader creates a new ganged fader from multiple channels
// levelControls are optional read-only controls for signal level indication
// taperDb specifies the dB range for DecibelTaper; if 0, LinearTaper is used
func NewGangedFader(name, unit string, mode GangMode, channels []*MixerChannel, levelControls []*scarlettctl.Control, taperDb float32) (*GangedFader, error) {
	if len(channels) < 1 {
		return nil, fmt.Errorf("ganged fader must have at least 1 channels")
	}

	// For mirror mode, use the first channel's range
	// Assume all channels have the same range (typical for mixer controls)
	firstControl := channels[0].GetControl()
	min := firstControl.Min
	max := firstControl.Max

	// Read initial value from first channel
	initialValue := channels[0].GetCurrentValue()

	gf := &GangedFader{
		name:          name,
		unit:          unit,
		mode:          mode,
		channels:      channels,
		lastValue:     initialValue,
		min:           min,
		max:           max,
		taperDb:       taperDb,
		levelControls: levelControls,
	}

	// Get level control range from first level control (if any)
	if len(levelControls) > 0 {
		gf.levelMin = levelControls[0].Min
		gf.levelMax = levelControls[0].Max
	}

	// Configure fader parameters
	gf.params = gf.createFaderParams()

	return gf, nil
}

// createFaderParams creates dfx.FaderParams for the ganged fader
func (gf *GangedFader) createFaderParams() dfx.FaderParams {
	var taper dfx.Taper
	if gf.taperDb > 0 {
		taper = dfx.DecibelTaper(gf.taperDb)
	} else {
		taper = dfx.LinearTaper()
	}

	params := dfx.FaderParams{
		Width:       60.0,
		Height:      300.0,
		ShowTooltip: true,
		Taper:       taper,
	}

	// Configure display format based on unit
	switch gf.unit {
	case "db":
		// Scarlett mixer control dB conversion: logarithmic scale from -∞ to +12 dB
		// This matches the formula used in alsa-scarlett-gui for mixer volumes
		params.Format = func(normalized float32) string {
			min := float32(gf.min)
			max := float32(gf.max)
			rawValue := normalized*(max-min) + min

			// Handle mute/zero case
			if rawValue <= min {
				return "-∞ dB"
			}

			// Logarithmic conversion: 0 to max maps to -∞ to +12 dB
			db := 20.0*math.Log10(float64(rawValue)/float64(gf.max)) + 12.0
			return fmt.Sprintf("%.2f dB", db)
		}
	case "raw":
		fallthrough
	default:
		params.Format = func(normalized float32) string {
			min := float32(gf.min)
			max := float32(gf.max)
			value := int(normalized*(max-min) + min)
			return fmt.Sprintf("%d", value)
		}
	}

	return params
}

// HandleUIChange is called when the user changes the ganged fader
// Writes to all ganged channels based on the gang mode
func (gf *GangedFader) HandleUIChange(newValue int64) error {
	// Value equality check
	oldValue := atomic.LoadInt64(&gf.lastValue)
	if oldValue == newValue {
		return nil
	}

	// Update cached value
	atomic.StoreInt64(&gf.lastValue, newValue)

	// Write to all ganged channels based on mode
	switch gf.mode {
	case GangModeMirror:
		return gf.handleMirrorMode(newValue)

	case GangModeRelative:
		// Future: implement relative mode (maintains offsets)
		log.Printf("Relative gang mode not yet implemented, using mirror mode")
		return gf.handleMirrorMode(newValue)

	case GangModeScaled:
		// Future: implement scaled mode
		log.Printf("Scaled gang mode not yet implemented, using mirror mode")
		return gf.handleMirrorMode(newValue)

	default:
		return fmt.Errorf("unknown gang mode: %s", gf.mode)
	}
}

// handleMirrorMode writes the same value to all ganged channels
func (gf *GangedFader) handleMirrorMode(value int64) error {
	var lastErr error

	for _, ch := range gf.channels {
		// Write to each channel - HandleUIChange has its own equality check
		if err := ch.HandleUIChange(value); err != nil {
			log.Printf("Failed to write to %s: %v", ch.GetDisplayName(), err)
			lastErr = err
		}
	}

	return lastErr
}

// HandleHWChange is called when one of the ganged hardware controls changes
// This is called by the event monitor when a ganged control changes externally
func (gf *GangedFader) HandleHWChange(numID uint, newValue int64) {
	// Find which channel changed
	for _, ch := range gf.channels {
		if ch.GetControl().NumID == numID {
			// Update that channel's cached value
			ch.HandleHWChange(newValue)

			// For mirror mode, also update our ganged fader value
			// Use the new value from the changed channel
			if gf.mode == GangModeMirror {
				atomic.StoreInt64(&gf.lastValue, newValue)
			}

			break
		}
	}
}

// GetCurrentValue returns the current cached value
func (gf *GangedFader) GetCurrentValue() int64 {
	return atomic.LoadInt64(&gf.lastValue)
}

// GetName returns the display name
func (gf *GangedFader) GetName() string {
	return gf.name
}

// GetParams returns the fader parameters
func (gf *GangedFader) GetParams() dfx.FaderParams {
	return gf.params
}

// GetMin returns the minimum value
func (gf *GangedFader) GetMin() int64 {
	return gf.min
}

// GetMax returns the maximum value
func (gf *GangedFader) GetMax() int64 {
	return gf.max
}

// GetChannels returns the ganged channels
func (gf *GangedFader) GetChannels() []*MixerChannel {
	return gf.channels
}

// HasLevels returns true if this gang has level controls configured
func (gf *GangedFader) HasLevels() bool {
	return len(gf.levelControls) > 0
}

// GetMaxLevel reads all level controls and returns the maximum value
// Returns the level value and true if successful, or 0 and false if no levels configured
func (gf *GangedFader) GetMaxLevel() (int64, bool) {
	if len(gf.levelControls) == 0 {
		return 0, false
	}

	var maxLevel int64
	for _, ctl := range gf.levelControls {
		val, err := ctl.GetValue()
		if err == nil && val > maxLevel {
			maxLevel = val
		}
	}
	return maxLevel, true
}

// GetLevelColor computes the track color based on current signal level
// Returns nil if no level controls are configured
// Color gradient: black (zero) -> dark green (low) -> bright green -> yellow -> red (high)
// Uses logarithmic (dB) scale for more sensitivity at lower levels
func (gf *GangedFader) GetLevelColor() *imgui.Vec4 {
	level, ok := gf.GetMaxLevel()
	if !ok {
		return nil
	}

	// Zero level = black
	if level == 0 {
		return &imgui.Vec4{X: 0, Y: 0, Z: 0, W: 1.0}
	}

	// Normalize to 0.0-1.0 using logarithmic (dB) scale
	// This provides much more sensitivity at lower signal levels
	var normalized float32
	if level <= gf.levelMin || gf.levelMax <= 0 {
		normalized = 0
	} else {
		// Convert to dB scale: 20 * log10(level / max)
		// This gives us 0 dB at max, negative values below
		ratio := float64(level) / float64(gf.levelMax)
		db := 20.0 * math.Log10(ratio)

		// Use 96 dB range (16-bit dynamic range) for more sensitivity at low levels
		// -96 dB -> 0.0, 0 dB -> 1.0
		const dbRange = 96.0
		if db < -dbRange {
			db = -dbRange
		}
		normalized = float32((db + dbRange) / dbRange)
	}

	if normalized < 0 {
		normalized = 0
	} else if normalized > 1 {
		normalized = 1
	}

	// Compute color using HSV
	// 0%: dark green (H=120°, S=1, V=0.3)
	// 50%: bright green (H=120°, S=1, V=0.6)
	// 80%: yellow (H=60°, S=1, V=0.8)
	// 100%: red (H=0°, S=1, V=1.0)
	var h, s, v float32
	s = 1.0

	if normalized <= 0.5 {
		// 0-50%: dark green to bright green (increase V)
		h = 120.0 / 360.0
		v = 0.3 + (normalized/0.5)*0.3 // 0.3 to 0.6
	} else if normalized <= 0.8 {
		// 50-80%: green to yellow (H from 120 to 60)
		t := (normalized - 0.5) / 0.3
		h = (120.0 - t*60.0) / 360.0 // 120° to 60°
		v = 0.6 + t*0.2              // 0.6 to 0.8
	} else {
		// 80-100%: yellow to red (H from 60 to 0)
		t := (normalized - 0.8) / 0.2
		h = (60.0 - t*60.0) / 360.0 // 60° to 0°
		v = 0.8 + t*0.2             // 0.8 to 1.0
	}

	var r, g, b float32
	imgui.ColorConvertHSVtoRGB(h, s, v, &r, &g, &b)

	return &imgui.Vec4{X: r, Y: g, Z: b, W: 1.0}
}
