package session

import (
	"github.com/michaelquigley/scarlettctl"
	"github.com/michaelquigley/sessionmixer/topology"
)

// Session is the complete resolved session containing all devices and cue mixes.
type Session struct {
	// Config is the original configuration
	Config *SessionConfig

	// TopologyDevice is the hardware topology
	TopologyDevice *topology.Device

	// State is the runtime state (levels, etc.)
	State *topology.DeviceState

	// Card is the ALSA card reference
	Card *scarlettctl.Card

	// Devices indexed by name
	Devices map[string]*Device

	// DeviceList in configuration order
	DeviceList []*Device

	// CueMixes in configuration order
	CueMixes []*CueMix
}

// GetDevice returns a device by name, or nil if not found
func (s *Session) GetDevice(name string) *Device {
	return s.Devices[name]
}

// Device is the resolved runtime representation of a DeviceConfig.
// It maps a named audio source to topology ports.
type Device struct {
	// Config is the original device configuration
	Config *DeviceConfig

	// Ports are the resolved topology ports (1 for mono, 2 for stereo)
	Ports []*topology.Port

	// IsStereo is true if this device has two ports
	IsStereo bool

	// LevelMeterIndices are the level meter indices for these ports
	LevelMeterIndices []int
}

// Name returns the device display name
func (d *Device) Name() string {
	return d.Config.Name
}

// CueMix is the resolved runtime representation of a cue mix.
// It contains device faders and controls routing to outputs.
type CueMix struct {
	// Config is the original cue mix configuration
	Config *CueMixConfig

	// Session reference
	Session *Session

	// MixL is the primary (left) mix
	MixL *topology.Mix

	// MixR is the secondary (right) mix for stereo, nil for mono
	MixR *topology.Mix

	// IsStereo is true if this cue mix uses a stereo pair
	IsStereo bool

	// Outputs are the hardware output ports
	Outputs []*topology.Port

	// Endpoints are the routing endpoints for the outputs
	Endpoints []*topology.RoutingEndpoint

	// Faders for each device in this cue mix
	Faders []*DeviceFader

	// Mute state and saved routing are managed in cuemix.go
}

// Name returns the cue mix display name
func (cm *CueMix) Name() string {
	return cm.Config.Name
}

// DeviceFader controls a device's level in a cue mix.
// For stereo cue mixes, it gangs the L and R mix inputs together.
type DeviceFader struct {
	// Device is the source device
	Device *Device

	// CueMix is the parent cue mix
	CueMix *CueMix

	// MixInputsL are the mix inputs in MixL for this device
	// One input for mono device, two for stereo device
	MixInputsL []*topology.MixInput

	// MixInputsR are the mix inputs in MixR for this device (nil for mono cue mix)
	MixInputsR []*topology.MixInput

	// VolumeControls are all the ALSA volume controls to gang together
	// This includes both L and R mix inputs for stereo cue mixes
	VolumeControls []*scarlettctl.Control

	// lastValue is the cached fader value for bidirectional updates (atomic)
	lastValue int64
}

// Name returns the device name for display
func (df *DeviceFader) Name() string {
	return df.Device.Name()
}
