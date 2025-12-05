package sessionmixer

import (
	"log"

	"github.com/michaelquigley/scarlettctl"
)

// EventMonitor handles hardware change events from scarlettctl
// Implements the Hardware → UI flow in the bidirectional update strategy
type EventMonitor struct {
	card    *scarlettctl.Card
	gangs   []*GangedFader
	monitor *scarlettctl.EventMonitor
}

// NewEventMonitor creates a new event monitor
func NewEventMonitor(card *scarlettctl.Card, gangs []*GangedFader) *EventMonitor {
	return &EventMonitor{
		card:    card,
		gangs:   gangs,
		monitor: card.NewEventMonitor(),
	}
}

// Start begins monitoring hardware events in a background goroutine
// This is event-driven, not polling (per BIDIRECTIONAL_UPDATE_STRATEGY.md)
func (em *EventMonitor) Start() error {
	// Start watching for control changes in a goroutine
	// WatchControls is blocking, so we run it in the background
	go func() {
		err := em.monitor.WatchControls(em.handleControlChange)
		if err != nil {
			log.Printf("Event monitor error: %v", err)
		}
	}()
	return nil
}

// Stop stops the event monitor
func (em *EventMonitor) Stop() {
	em.monitor.Stop()
}

// handleControlChange is the callback invoked when a hardware control changes
// This is called from the scarlettctl event monitor goroutine
// It uses thread-safe atomic operations to update cached values
func (em *EventMonitor) handleControlChange(control *scarlettctl.Control, value int64) error {
	// Check if this control belongs to a ganged fader
	for _, gang := range em.gangs {
		for _, ch := range gang.GetChannels() {
			if ch.GetControl().NumID == control.NumID {
				// Update the gang's cached value
				// HandleHWChange has value equality check
				gang.HandleHWChange(control.NumID, value)
				return nil
			}
		}
	}

	// Control not found in our configuration (this is okay - we might not be
	// monitoring all controls on the card)
	return nil
}

// GetMonitor returns the underlying scarlettctl event monitor
func (em *EventMonitor) GetMonitor() *scarlettctl.EventMonitor {
	return em.monitor
}
