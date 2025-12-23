package session

import (
	"log"
	"sync/atomic"
)

// Bidirectional update methods for DeviceFader.
// Implements the strategy from BIDIRECTIONAL_UPDATE_STRATEGY.md.

// lastValue stores the cached fader value for bidirectional updates
// This is stored in DeviceFader struct as an int64 field

// InitializeFromHardware reads the initial value from hardware and zeros cross inputs.
// Should be called after building the session.
func (df *DeviceFader) InitializeFromHardware() error {
	if len(df.VolumeControls) == 0 {
		return nil
	}

	// Read from the first control (they should all be synced)
	val, err := df.VolumeControls[0].GetValue()
	if err != nil {
		return err
	}

	atomic.StoreInt64(&df.lastValue, val)

	// Zero out cross inputs for proper stereo separation
	// (these are the "wrong" channels: L source in R mix, R source in L mix)
	for _, crossInput := range df.CrossInputs {
		if crossInput.VolumeControl != nil {
			if err := crossInput.VolumeControl.SetValue(crossInput.VolumeControl.Min); err != nil {
				log.Printf("Failed to zero cross input %s: %v", crossInput.VolumeControl.Name, err)
			}
		}
	}

	return nil
}

// GetCurrentValue returns the current cached value (thread-safe)
func (df *DeviceFader) GetCurrentValue() int64 {
	return atomic.LoadInt64(&df.lastValue)
}

// Min returns the minimum value for the fader
func (df *DeviceFader) Min() int64 {
	if len(df.VolumeControls) == 0 {
		return 0
	}
	return df.VolumeControls[0].Min
}

// Max returns the maximum value for the fader
func (df *DeviceFader) Max() int64 {
	if len(df.VolumeControls) == 0 {
		return 0
	}
	return df.VolumeControls[0].Max
}

// HandleUIChange is called when the user changes the fader in the UI.
// It writes the new value to all ganged volume controls.
// Implements immediate write with value equality check.
func (df *DeviceFader) HandleUIChange(newValue int64) error {
	// Value equality check - skip if unchanged
	oldValue := atomic.LoadInt64(&df.lastValue)
	if oldValue == newValue {
		return nil
	}

	// Update cached value
	atomic.StoreInt64(&df.lastValue, newValue)

	// Write to ALL ganged volume controls
	for _, ctrl := range df.VolumeControls {
		if err := ctrl.SetValue(newValue); err != nil {
			log.Printf("Failed to write to %s: %v", ctrl.Name, err)
			// Continue writing to other controls even if one fails
		}
	}

	return nil
}

// HandleHWChange is called when hardware state changes (from event monitor).
// Implements value equality check to prevent feedback loops.
func (df *DeviceFader) HandleHWChange(numID uint, newValue int64) {
	// Check if this numID belongs to one of our controls
	found := false
	for _, ctrl := range df.VolumeControls {
		if ctrl.NumID == numID {
			found = true
			break
		}
	}
	if !found {
		return
	}

	// Value equality check - break feedback loop
	oldValue := atomic.LoadInt64(&df.lastValue)
	if oldValue == newValue {
		return
	}

	// Update cached value (hardware is source of truth)
	atomic.StoreInt64(&df.lastValue, newValue)

	// The next Draw() will use this new value automatically
}

// GetNumIDs returns all control NumIDs for event monitoring registration
func (df *DeviceFader) GetNumIDs() []uint {
	numIDs := make([]uint, 0, len(df.VolumeControls))
	for _, ctrl := range df.VolumeControls {
		numIDs = append(numIDs, ctrl.NumID)
	}
	return numIDs
}
