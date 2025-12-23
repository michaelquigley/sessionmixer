package sessionmixer

import (
	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/michaelquigley/dfx"
	"github.com/michaelquigley/sessionmixer/session"
)

// SessionMixer is the main mixer component
// Implements dfx.Component interface for immediate-mode GUI rendering
type SessionMixer struct {
	session   *session.Session
	cueMixUIs []*CueMixUI
	monitor   *SessionEventMonitor
}

// NewSessionMixer creates a new session mixer from a session.
// waterfallStates contains saved waterfall open/closed states keyed by "cueMixName/deviceName" or "cueMixName/master".
func NewSessionMixer(sess *session.Session, waterfallStates map[string]bool) *SessionMixer {
	sm := &SessionMixer{
		session: sess,
	}

	// Ensure waterfallStates is not nil
	if waterfallStates == nil {
		waterfallStates = make(map[string]bool)
	}

	// Create UI components for each cue mix
	levelSmoothing := sess.Config.LevelSmoothing
	for _, cueMix := range sess.CueMixes {
		cueMixUI := NewCueMixUI(cueMix, sess.State, levelSmoothing, waterfallStates)
		sm.cueMixUIs = append(sm.cueMixUIs, cueMixUI)
	}

	return sm
}

// Draw renders the mixer UI using dfx immediate mode
// This is called every frame by the dfx application
func (sm *SessionMixer) Draw(state *dfx.State) {
	if len(sm.cueMixUIs) == 0 {
		imgui.Text("No cue mixes configured")
		imgui.Text("")
		imgui.Text("Add cue_mixes to your session.yaml configuration.")
		return
	}

	// Draw each cue mix section
	for _, cueMixUI := range sm.cueMixUIs {
		cueMixUI.Draw(state)
		imgui.Spacing()
	}
}

// Actions returns the action registry for keyboard shortcuts
func (sm *SessionMixer) Actions() *dfx.ActionRegistry {
	return nil // No custom actions for now
}

// SetMonitor sets the event monitor for hardware change notifications
func (sm *SessionMixer) SetMonitor(monitor *SessionEventMonitor) {
	sm.monitor = monitor
}

// GetSession returns the session
func (sm *SessionMixer) GetSession() *session.Session {
	return sm.session
}

// GetCueMixUIs returns the cue mix UI components
func (sm *SessionMixer) GetCueMixUIs() []*CueMixUI {
	return sm.cueMixUIs
}

// GetAllDeviceFaders returns all device faders from all cue mixes
// This is used for event monitor registration
func (sm *SessionMixer) GetAllDeviceFaders() []*session.DeviceFader {
	var faders []*session.DeviceFader
	for _, cueMixUI := range sm.cueMixUIs {
		for _, faderUI := range cueMixUI.GetFaders() {
			faders = append(faders, faderUI.GetFader())
		}
	}
	return faders
}

// GetWaterfallStates returns the current waterfall open/closed states for all channels.
// Keys are "cueMixName/deviceName" or "cueMixName/master".
func (sm *SessionMixer) GetWaterfallStates() map[string]bool {
	states := make(map[string]bool)
	for _, cueMixUI := range sm.cueMixUIs {
		for key, value := range cueMixUI.GetWaterfallStates() {
			states[key] = value
		}
	}
	return states
}
