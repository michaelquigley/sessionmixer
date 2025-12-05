package sessionmixer

import (
	"fmt"

	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/michaelquigley/dfx"
	"github.com/michaelquigley/scarlettctl"
)

// SessionMixer is the main mixer component
// Implements dfx.Component interface for immediate-mode GUI rendering
type SessionMixer struct {
	card    *scarlettctl.Card
	config  *Config
	gangs   []*GangedFader
	monitor *EventMonitor
}

// NewSessionMixer creates a new session mixer
func NewSessionMixer(card *scarlettctl.Card, config *Config, gangs []*GangedFader) *SessionMixer {
	return &SessionMixer{
		card:   card,
		config: config,
		gangs:  gangs,
	}
}

// Draw renders the mixer UI using dfx immediate mode
// This is called every frame by the dfx application
func (sm *SessionMixer) Draw(_ *dfx.State) {
	// Calculate total number of faders (individual channels + gangs)
	totalFaders := len(sm.gangs)

	if totalFaders == 0 {
		imgui.Text("No controls configured")
		return
	}

	imgui.Dummy(imgui.Vec2{X: 25, Y: 100})
	imgui.SameLine()

	// Create scrollable child window for fader bank
	// Similar to dfx_example_mixer layout
	childSize := imgui.Vec2{X: 0, Y: 450} // X=0 fills available width
	imgui.BeginChildStrV("FaderBank", childSize,
		imgui.ChildFlagsNone,
		imgui.WindowFlagsHorizontalScrollbar)

	// Use table layout for stable column widths
	faderWidth := float32(80.0) // Width per fader column
	contentWidth := float32(totalFaders) * faderWidth

	imgui.BeginTableV("mixer_table", int32(totalFaders),
		imgui.TableFlagsNone,
		imgui.Vec2{X: contentWidth, Y: 0}, 0.0)

	// Setup fixed-width columns
	for i := 0; i < totalFaders; i++ {
		imgui.TableSetupColumnV(fmt.Sprintf("##col%d", i),
			imgui.TableColumnFlagsWidthFixed, faderWidth, 0)
	}

	// Row 1: Channel labels
	imgui.TableNextRow()
	for _, gang := range sm.gangs {
		imgui.TableNextColumn()
		imgui.Text(gang.GetName())
	}

	// Row 2: Faders
	imgui.TableNextRow()

	// Draw ganged faders
	for i, gang := range sm.gangs {
		imgui.TableNextColumn()

		currentValue := int(gang.GetCurrentValue())

		// Get params and set TrackColor if gang has level controls
		params := gang.GetParams()
		if gang.HasLevels() {
			params.TrackColor = gang.GetLevelColor()
		}

		// Use dfx.FaderI for ganged fader
		newValue, changed := dfx.FaderI(
			fmt.Sprintf("##fader_gang_%d", i),
			currentValue,
			int(gang.GetMin()),
			int(gang.GetMax()),
			params)

		if changed {
			// IMMEDIATE write to all ganged channels
			gang.HandleUIChange(int64(newValue))
		}
	}

	// Row 3: Value displays
	imgui.TableNextRow()
	for _, gang := range sm.gangs {
		imgui.TableNextColumn()
		currentValue := gang.GetCurrentValue()
		imgui.Text(fmt.Sprintf("%d", currentValue))
	}

	imgui.EndTable()
	imgui.EndChild()
}

// Actions returns the action registry for keyboard shortcuts
func (sm *SessionMixer) Actions() *dfx.ActionRegistry {
	return nil // No custom actions for now
}

// SetMonitor sets the event monitor for hardware change notifications
func (sm *SessionMixer) SetMonitor(monitor *EventMonitor) {
	sm.monitor = monitor
}

// GetCard returns the scarlettctl card
func (sm *SessionMixer) GetCard() *scarlettctl.Card {
	return sm.card
}

// GetGangs returns the ganged faders
func (sm *SessionMixer) GetGangs() []*GangedFader {
	return sm.gangs
}
