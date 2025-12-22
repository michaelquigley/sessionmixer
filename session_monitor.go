package sessionmixer

import (
	"log"

	"github.com/michaelquigley/scarlettctl"
	"github.com/michaelquigley/sessionmixer/session"
)

// SessionEventMonitor handles hardware change events for session-based faders.
// Implements the Hardware → UI flow in the bidirectional update strategy.
type SessionEventMonitor struct {
	card    *scarlettctl.Card
	faders  []*session.DeviceFader
	monitor *scarlettctl.EventMonitor

	// Map from control NumID to fader for quick lookup
	fadersByNumID map[uint]*session.DeviceFader
}

// NewSessionEventMonitor creates a new event monitor for session-based faders.
func NewSessionEventMonitor(card *scarlettctl.Card, faders []*session.DeviceFader) *SessionEventMonitor {
	em := &SessionEventMonitor{
		card:          card,
		faders:        faders,
		monitor:       card.NewEventMonitor(),
		fadersByNumID: make(map[uint]*session.DeviceFader),
	}

	// Build lookup table
	for _, fader := range faders {
		for _, numID := range fader.GetNumIDs() {
			em.fadersByNumID[numID] = fader
		}
	}

	return em
}

// Start begins monitoring hardware events in a background goroutine.
// This is event-driven, not polling.
func (em *SessionEventMonitor) Start() error {
	go func() {
		err := em.monitor.WatchControls(em.handleControlChange)
		if err != nil {
			log.Printf("Session event monitor error: %v", err)
		}
	}()
	return nil
}

// Stop stops the event monitor.
func (em *SessionEventMonitor) Stop() {
	em.monitor.Stop()
}

// handleControlChange is the callback invoked when a hardware control changes.
// This is called from the scarlettctl event monitor goroutine.
// It uses thread-safe atomic operations to update cached values.
func (em *SessionEventMonitor) handleControlChange(control *scarlettctl.Control, value int64) error {
	// Look up fader by control NumID
	if fader, ok := em.fadersByNumID[control.NumID]; ok {
		// Update the fader's cached value
		// HandleHWChange has value equality check
		fader.HandleHWChange(control.NumID, value)
	}

	// Control not found in our configuration is okay - we might not be
	// monitoring all controls on the card
	return nil
}

// GetMonitor returns the underlying scarlettctl event monitor.
func (em *SessionEventMonitor) GetMonitor() *scarlettctl.EventMonitor {
	return em.monitor
}
