package sessionmixer

import (
	"fmt"
	"log"
	"math"
	"sync/atomic"

	"github.com/michaelquigley/scarlettctl"
)

// MixerChannel represents a single fader control in the mixer
// Implements bidirectional update strategy from BIDIRECTIONAL_UPDATE_STRATEGY.md
type MixerChannel struct {
	// Hardware control
	control *scarlettctl.Control

	// Display properties
	displayName string
	unit        string

	// Cached values (thread-safe via atomic operations)
	// These are caches, not authoritative state - hardware is source of truth
	lastUIValue int64 // Last value set BY the UI
	lastHWValue int64 // Last value FROM hardware
}

// NewMixerChannel creates a new mixer channel from a hardware control
func NewMixerChannel(control *scarlettctl.Control, displayName, unit string) (*MixerChannel, error) {
	if control == nil {
		return nil, fmt.Errorf("control cannot be nil")
	}

	// Read initial value from hardware
	initialValue, err := control.GetValue()
	if err != nil {
		return nil, fmt.Errorf("failed to read initial value: %w", err)
	}

	ch := &MixerChannel{
		control:     control,
		displayName: displayName,
		unit:        unit,
		lastUIValue: initialValue,
		lastHWValue: initialValue,
	}

	return ch, nil
}

// HandleUIChange is called when the user changes the fader in the UI
// Implements immediate write with value equality check (no debouncing)
// This is part of the UI → Hardware flow in the bidirectional update strategy
func (ch *MixerChannel) HandleUIChange(newValue int64) error {
	// CRITICAL: Value equality check - skip if unchanged
	// This prevents redundant writes when dragging
	oldValue := atomic.LoadInt64(&ch.lastUIValue)
	if oldValue == newValue {
		return nil // No change, don't write
	}

	// Update cached value
	atomic.StoreInt64(&ch.lastUIValue, newValue)

	// IMMEDIATE write to hardware - no debouncing, no delay
	// The ALSA driver will handle batching rapid updates naturally
	err := ch.control.SetValue(newValue)
	if err != nil {
		log.Printf("Failed to write to %s: %v", ch.control.Name, err)
		return err
	}

	return nil
}

// HandleHWChange is called when hardware state changes (from event monitor)
// Implements value equality check to prevent feedback loops
// This is part of the Hardware → UI flow in the bidirectional update strategy
func (ch *MixerChannel) HandleHWChange(newValue int64) {
	// CRITICAL: Value equality check - skip if unchanged
	// This is the KEY to preventing feedback loops:
	// When UI writes to hardware, hardware event fires with the SAME value,
	// we detect oldValue == newValue and return early, breaking the loop!
	oldValue := atomic.LoadInt64(&ch.lastHWValue)
	if oldValue == newValue {
		return // No actual change
	}

	// Update both cached values
	// Sync UI value to match hardware (hardware is source of truth)
	atomic.StoreInt64(&ch.lastHWValue, newValue)
	atomic.StoreInt64(&ch.lastUIValue, newValue)

	// The next Draw() call will use this new value automatically
	// No need to explicitly trigger UI update in immediate mode
}

// GetCurrentValue returns the current cached value (thread-safe)
func (ch *MixerChannel) GetCurrentValue() int64 {
	return atomic.LoadInt64(&ch.lastUIValue)
}

// GetControl returns the underlying hardware control
func (ch *MixerChannel) GetControl() *scarlettctl.Control {
	return ch.control
}

// GetDisplayName returns the display name for this channel
func (ch *MixerChannel) GetDisplayName() string {
	return ch.displayName
}

// ConvertToDB converts a raw hardware value to decibels
// This is a generic conversion - adjust for specific hardware
func (ch *MixerChannel) ConvertToDB(rawValue int64) float64 {
	if ch.control.Max == ch.control.Min {
		return 0.0
	}

	// Normalize to 0.0 - 1.0
	normalized := float64(rawValue-ch.control.Min) / float64(ch.control.Max-ch.control.Min)

	// Convert to dB (assuming -60dB to +12dB range)
	// Adjust this formula based on your specific hardware
	db := (normalized * 72.0) - 60.0

	return db
}

// ConvertFromDB converts decibels to raw hardware value
// This is a generic conversion - adjust for specific hardware
func (ch *MixerChannel) ConvertFromDB(db float64) int64 {
	// Clamp to range
	if db < -60.0 {
		db = -60.0
	}
	if db > 12.0 {
		db = 12.0
	}

	// Convert to normalized 0.0 - 1.0
	normalized := (db + 60.0) / 72.0

	// Convert to raw hardware value
	rawValue := int64(math.Round(normalized*float64(ch.control.Max-ch.control.Min))) + ch.control.Min

	return rawValue
}
