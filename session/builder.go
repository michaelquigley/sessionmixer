package session

import (
	"fmt"

	"github.com/michaelquigley/scarlettctl"
	"github.com/michaelquigley/sessionmixer/topology"
)

// SessionBuilder constructs a Session from configuration and topology.
type SessionBuilder struct {
	config *SessionConfig
	device *topology.Device
	state  *topology.DeviceState
	card   *scarlettctl.Card
}

// NewSessionBuilder creates a new builder with the given configuration and topology.
func NewSessionBuilder(config *SessionConfig, device *topology.Device, state *topology.DeviceState, card *scarlettctl.Card) *SessionBuilder {
	return &SessionBuilder{
		config: config,
		device: device,
		state:  state,
		card:   card,
	}
}

// Build creates the complete session by resolving all configuration to topology objects.
func (b *SessionBuilder) Build() (*Session, error) {
	session := &Session{
		Config:         b.config,
		TopologyDevice: b.device,
		State:          b.state,
		Card:           b.card,
		Devices:        make(map[string]*Device),
	}

	// Resolve devices
	for i := range b.config.Devices {
		deviceCfg := &b.config.Devices[i]
		device, err := b.resolveDevice(deviceCfg)
		if err != nil {
			return nil, fmt.Errorf("device %q: %w", deviceCfg.Name, err)
		}
		session.Devices[device.Name()] = device
		session.DeviceList = append(session.DeviceList, device)
	}

	// Resolve cue mixes
	for i := range b.config.CueMixes {
		cueMixCfg := &b.config.CueMixes[i]
		cueMix, err := b.resolveCueMix(cueMixCfg, session)
		if err != nil {
			return nil, fmt.Errorf("cue mix %q: %w", cueMixCfg.Name, err)
		}
		session.CueMixes = append(session.CueMixes, cueMix)
	}

	return session, nil
}

// resolveDevice resolves a DeviceConfig to a Device by looking up ports.
func (b *SessionBuilder) resolveDevice(cfg *DeviceConfig) (*Device, error) {
	if len(cfg.Ports) == 0 {
		return nil, fmt.Errorf("no ports specified")
	}
	if len(cfg.Ports) > 2 {
		return nil, fmt.Errorf("device can have at most 2 ports (stereo)")
	}

	device := &Device{
		Config:   cfg,
		IsStereo: len(cfg.Ports) == 2,
	}

	for _, portID := range cfg.Ports {
		port := b.device.GetPort(portID)
		if port == nil {
			return nil, fmt.Errorf("port %q not found", portID)
		}
		device.Ports = append(device.Ports, port)

		// Collect level meter index if available
		if port.LevelMeterIndex >= 0 {
			device.LevelMeterIndices = append(device.LevelMeterIndices, port.LevelMeterIndex)
		}
	}

	return device, nil
}

// resolveCueMix resolves a CueMixConfig to a CueMix.
func (b *SessionBuilder) resolveCueMix(cfg *CueMixConfig, session *Session) (*CueMix, error) {
	// Parse mix pair
	leftLetter, rightLetter, isStereo := cfg.ParseMixPair()
	if leftLetter == 0 {
		return nil, fmt.Errorf("invalid mix_pair %q", cfg.MixPair)
	}

	cueMix := &CueMix{
		Config:   cfg,
		Session:  session,
		IsStereo: isStereo,
	}

	// Resolve left mix
	cueMix.MixL = b.device.GetMixByLetter(leftLetter)
	if cueMix.MixL == nil {
		return nil, fmt.Errorf("mix %c not found", leftLetter)
	}

	// Resolve right mix if stereo
	if isStereo {
		cueMix.MixR = b.device.GetMixByLetter(rightLetter)
		if cueMix.MixR == nil {
			return nil, fmt.Errorf("mix %c not found", rightLetter)
		}
	}

	// Resolve output ports
	for _, portID := range cfg.Outputs {
		port := b.device.GetPort(portID)
		if port == nil {
			return nil, fmt.Errorf("output port %q not found", portID)
		}
		cueMix.Outputs = append(cueMix.Outputs, port)

		// Get routing endpoint for this output
		endpoint := b.device.GetRoutingEndpointForPort(port)
		if endpoint == nil {
			return nil, fmt.Errorf("no routing endpoint for output %q", portID)
		}
		cueMix.Endpoints = append(cueMix.Endpoints, endpoint)
	}

	// Create device faders
	for _, deviceName := range cfg.Devices {
		device := session.GetDevice(deviceName)
		if device == nil {
			return nil, fmt.Errorf("device %q not found", deviceName)
		}

		fader, err := b.createDeviceFader(device, cueMix)
		if err != nil {
			return nil, fmt.Errorf("device %q fader: %w", deviceName, err)
		}
		cueMix.Faders = append(cueMix.Faders, fader)
	}

	return cueMix, nil
}

// createDeviceFader creates a DeviceFader for a device in a cue mix.
func (b *SessionBuilder) createDeviceFader(device *Device, cueMix *CueMix) (*DeviceFader, error) {
	fader := &DeviceFader{
		Device: device,
		CueMix: cueMix,
	}

	// Find mix inputs for each device port in MixL
	for _, port := range device.Ports {
		mixInput := b.findMixInputForPort(cueMix.MixL, port)
		if mixInput == nil {
			return nil, fmt.Errorf("no mix input found for port %s in mix %s", port.ID, cueMix.MixL.ID)
		}
		fader.MixInputsL = append(fader.MixInputsL, mixInput)
		if mixInput.VolumeControl != nil {
			fader.VolumeControls = append(fader.VolumeControls, mixInput.VolumeControl)
		}
	}

	// For stereo cue mix, also find inputs in MixR
	if cueMix.IsStereo && cueMix.MixR != nil {
		for _, port := range device.Ports {
			mixInput := b.findMixInputForPort(cueMix.MixR, port)
			if mixInput == nil {
				return nil, fmt.Errorf("no mix input found for port %s in mix %s", port.ID, cueMix.MixR.ID)
			}
			fader.MixInputsR = append(fader.MixInputsR, mixInput)
			if mixInput.VolumeControl != nil {
				fader.VolumeControls = append(fader.VolumeControls, mixInput.VolumeControl)
			}
		}
	}

	if len(fader.VolumeControls) == 0 {
		return nil, fmt.Errorf("no volume controls found")
	}

	return fader, nil
}

// findMixInputForPort finds the MixInput in a mix that has the given port as its source.
func (b *SessionBuilder) findMixInputForPort(mix *topology.Mix, port *topology.Port) *topology.MixInput {
	for _, input := range mix.Inputs {
		if input.Source == port {
			return input
		}
	}
	return nil
}
