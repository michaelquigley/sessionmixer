package session

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
)

// Mute state and routing management for CueMix.

// muted is stored as int32 in CueMix struct (0 = unmuted, 1 = muted)
// savedRouting is stored as []string in CueMix struct

// CueMixState holds mutable state for a CueMix
type CueMixState struct {
	muted        int32    // atomic: 0=unmuted, 1=muted
	savedRouting []string // Saved routing sources when muted
	mu           sync.Mutex
}

// cueMixStates stores state for each CueMix by pointer
var cueMixStates = make(map[*CueMix]*CueMixState)
var cueMixStatesMu sync.RWMutex

// getState returns the state for a CueMix, creating if needed
func (cm *CueMix) getState() *CueMixState {
	cueMixStatesMu.RLock()
	state := cueMixStates[cm]
	cueMixStatesMu.RUnlock()

	if state != nil {
		return state
	}

	cueMixStatesMu.Lock()
	defer cueMixStatesMu.Unlock()

	// Double-check after acquiring write lock
	if state = cueMixStates[cm]; state != nil {
		return state
	}

	state = &CueMixState{
		savedRouting: make([]string, len(cm.Endpoints)),
	}
	cueMixStates[cm] = state
	return state
}

// IsMuted returns true if this cue mix is muted
func (cm *CueMix) IsMuted() bool {
	state := cm.getState()
	return atomic.LoadInt32(&state.muted) != 0
}

// SetMuted sets the mute state for this cue mix.
// When muted, outputs are routed to "Off".
// When unmuted, outputs are restored to their saved routing.
func (cm *CueMix) SetMuted(muted bool) error {
	state := cm.getState()
	state.mu.Lock()
	defer state.mu.Unlock()

	currentMuted := atomic.LoadInt32(&state.muted) != 0
	if currentMuted == muted {
		return nil // No change
	}

	if muted {
		// Save current routing and set to "Off"
		for i, ep := range cm.Endpoints {
			state.savedRouting[i] = ep.GetCurrentSource()

			offIdx := findSourceIndex(ep.AvailableSources, "Off")
			if offIdx < 0 {
				log.Printf("Warning: 'Off' not found in routing sources for %s", ep.Port.Name)
				continue
			}

			if err := ep.RoutingControl.SetValue(int64(offIdx)); err != nil {
				return fmt.Errorf("failed to set routing for %s: %w", ep.Port.Name, err)
			}
			// Update cached state to match hardware
			ep.SetSourceIndex(offIdx)
		}
		atomic.StoreInt32(&state.muted, 1)
	} else {
		// Restore saved routing
		for i, ep := range cm.Endpoints {
			sourceName := state.savedRouting[i]
			if sourceName == "" {
				continue
			}

			idx := findSourceIndex(ep.AvailableSources, sourceName)
			if idx < 0 {
				log.Printf("Warning: saved source %q not found for %s", sourceName, ep.Port.Name)
				continue
			}

			if err := ep.RoutingControl.SetValue(int64(idx)); err != nil {
				return fmt.Errorf("failed to restore routing for %s: %w", ep.Port.Name, err)
			}
			// Update cached state to match hardware
			ep.SetSourceIndex(idx)
		}
		atomic.StoreInt32(&state.muted, 0)
	}

	return nil
}

// ToggleMute toggles the mute state
func (cm *CueMix) ToggleMute() error {
	return cm.SetMuted(!cm.IsMuted())
}

// SetupRouting configures the output routing for this cue mix.
// Routes outputs to the appropriate mix outputs (Mix A, Mix B, etc.).
func (cm *CueMix) SetupRouting() error {
	if len(cm.Endpoints) == 0 {
		return nil
	}

	// For stereo cue mix: route L output to Mix L, R output to Mix R
	// For mono cue mix: route output to Mix L
	sources := cm.getRoutingSources()

	for i, ep := range cm.Endpoints {
		if i >= len(sources) {
			break
		}

		sourceName := sources[i]
		idx := findSourceIndex(ep.AvailableSources, sourceName)
		if idx < 0 {
			return fmt.Errorf("routing source %q not found for %s (available: %v)",
				sourceName, ep.Port.Name, ep.AvailableSources)
		}

		if err := ep.RoutingControl.SetValue(int64(idx)); err != nil {
			return fmt.Errorf("failed to set routing for %s: %w", ep.Port.Name, err)
		}

		// Update the endpoint's cached state
		ep.SetSourceIndex(idx)
	}

	return nil
}

// getRoutingSources returns the source names to route outputs to.
// For stereo: ["Mix A", "Mix B"]
// For mono: ["Mix A"]
func (cm *CueMix) getRoutingSources() []string {
	if cm.IsStereo && cm.MixR != nil {
		return []string{cm.MixL.Name, cm.MixR.Name}
	}
	return []string{cm.MixL.Name}
}

// GetMixOutputLevelIndices returns the level meter indices for the mix outputs.
func (cm *CueMix) GetMixOutputLevelIndices() []int {
	var indices []int
	if cm.MixL != nil && cm.MixL.OutputLevelMeterIndex >= 0 {
		indices = append(indices, cm.MixL.OutputLevelMeterIndex)
	}
	if cm.MixR != nil && cm.MixR.OutputLevelMeterIndex >= 0 {
		indices = append(indices, cm.MixR.OutputLevelMeterIndex)
	}
	return indices
}

// InitializeFaders initializes all device faders from hardware values.
func (cm *CueMix) InitializeFaders() error {
	for _, fader := range cm.Faders {
		if err := fader.InitializeFromHardware(); err != nil {
			log.Printf("Warning: failed to initialize fader %s: %v", fader.Name(), err)
		}
	}
	return nil
}

// InitializeFromHardware reads current routing state from hardware and sets
// the mute state accordingly. If routing is already set up, the mix is active.
// If routing is not present, the mix starts muted.
func (cm *CueMix) InitializeFromHardware() error {
	state := cm.getState()
	state.mu.Lock()
	defer state.mu.Unlock()

	// Get expected routing sources (e.g., ["Mix A", "Mix B"])
	expectedSources := cm.getRoutingSources()

	// Refresh routing state from hardware
	for _, ep := range cm.Endpoints {
		if err := ep.RefreshFromHardware(); err != nil {
			return fmt.Errorf("failed to read routing for %s: %w", ep.Port.Name, err)
		}
	}

	// Check if current routing matches expected
	routingMatches := true
	for i, ep := range cm.Endpoints {
		if i >= len(expectedSources) {
			break
		}
		currentSource := ep.GetCurrentSource()
		if currentSource != expectedSources[i] {
			routingMatches = false
			break
		}
	}

	if routingMatches {
		// Routing is already set up - mix is active
		atomic.StoreInt32(&state.muted, 0)
		// Store current routing in savedRouting for consistency
		for i, ep := range cm.Endpoints {
			state.savedRouting[i] = ep.GetCurrentSource()
		}
	} else {
		// Routing not present - start muted
		atomic.StoreInt32(&state.muted, 1)
		// Store expected routing so unmute will set it up correctly
		for i := range cm.Endpoints {
			if i < len(expectedSources) {
				state.savedRouting[i] = expectedSources[i]
			}
		}
	}

	return nil
}

// findSourceIndex finds the index of a source name in the available sources list.
// Returns -1 if not found.
func findSourceIndex(sources []string, name string) int {
	for i, s := range sources {
		if s == name {
			return i
		}
	}
	return -1
}
