package topology

import (
	"fmt"
	"strings"

	"github.com/michaelquigley/scarlettctl"
)

// DeviceBuilder constructs a Device from hardware discovery
type DeviceBuilder struct {
	card    *scarlettctl.Card
	profile DeviceProfile

	// Control lookup maps (populated once in Build)
	controlsByName map[string]*scarlettctl.Control
	controlsByID   map[string]*scarlettctl.Control
}

// NewDeviceBuilder creates a builder for the given card and profile
func NewDeviceBuilder(card *scarlettctl.Card, profile DeviceProfile) *DeviceBuilder {
	return &DeviceBuilder{
		card:    card,
		profile: profile,
	}
}

// Build discovers controls and builds the complete device topology
func (b *DeviceBuilder) Build() (*Device, error) {
	// Fetch all controls once and build lookup maps
	if err := b.buildControlIndex(); err != nil {
		return nil, fmt.Errorf("building control index: %w", err)
	}

	device := &Device{
		Name:    b.profile.Name(),
		Card:    b.card,
		Profile: b.profile,
		Ports:   make(map[string]*Port),
		MixByID: make(map[string]*Mix),
	}

	// Build ports
	if err := b.buildAnalogueInputs(device); err != nil {
		return nil, fmt.Errorf("building analogue inputs: %w", err)
	}
	if err := b.buildAnalogueOutputs(device); err != nil {
		return nil, fmt.Errorf("building analogue outputs: %w", err)
	}
	if err := b.buildSPDIFInputs(device); err != nil {
		return nil, fmt.Errorf("building S/PDIF inputs: %w", err)
	}
	if err := b.buildSPDIFOutputs(device); err != nil {
		return nil, fmt.Errorf("building S/PDIF outputs: %w", err)
	}
	if err := b.buildADATInputs(device); err != nil {
		return nil, fmt.Errorf("building ADAT inputs: %w", err)
	}
	if err := b.buildADATOutputs(device); err != nil {
		return nil, fmt.Errorf("building ADAT outputs: %w", err)
	}
	if err := b.buildPCMPlayback(device); err != nil {
		return nil, fmt.Errorf("building PCM playback: %w", err)
	}
	if err := b.buildPCMCapture(device); err != nil {
		return nil, fmt.Errorf("building PCM capture: %w", err)
	}

	// Build mixes
	if err := b.buildMixes(device); err != nil {
		return nil, fmt.Errorf("building mixes: %w", err)
	}

	// Build routing endpoints
	if err := b.buildRoutingEndpoints(device); err != nil {
		return nil, fmt.Errorf("building routing endpoints: %w", err)
	}

	// Build mixer input routing endpoints
	if err := b.buildMixerInputEndpoints(device); err != nil {
		return nil, fmt.Errorf("building mixer input endpoints: %w", err)
	}

	// Collect all level meter controls
	b.collectLevelMeters(device)

	return device, nil
}

func (b *DeviceBuilder) buildAnalogueInputs(device *Device) error {
	for i := 1; i <= b.profile.AnalogueInputCount(); i++ {
		portID := fmt.Sprintf("analogue-in-%d", i)

		port := &Port{
			ID:              portID,
			Name:            fmt.Sprintf("Analogue Input %d", i),
			ShortName:       fmt.Sprintf("An%d", i),
			Type:            PortTypeAnalogue,
			Direction:       PortDirectionInput,
			Number:          i,
			HasPhantom:      b.profile.HasPhantom(i),
			HasAir:          b.profile.HasAir(i),
			HasPad:          b.profile.HasPad(i),
			HasGain:         b.profile.HasGain(i),
			LevelMeterIndex: b.profile.GetLevelMeterIndex(portID),
		}

		// Find associated controls (don't fail if not found)
		if port.HasGain {
			port.GainControl = b.findControl(b.profile.GainControlName(i))
		}

		if port.HasPhantom {
			port.PhantomControl = b.findControl(b.profile.PhantomControlName(i))
		}

		if port.HasAir {
			port.AirControl = b.findControl(b.profile.AirControlName(i))
		}

		if port.HasPad {
			port.PadControl = b.findControl(b.profile.PadControlName(i))
		}

		// Level meter
		if port.LevelMeterIndex >= 0 {
			port.LevelMeterControl = b.findControl(b.profile.LevelMeterControlName(port.LevelMeterIndex))
		}

		device.Ports[portID] = port
		device.AnalogueInputs = append(device.AnalogueInputs, port)
	}

	return nil
}

func (b *DeviceBuilder) buildAnalogueOutputs(device *Device) error {
	for i := 1; i <= b.profile.AnalogueOutputCount(); i++ {
		portID := fmt.Sprintf("analogue-out-%d", i)

		port := &Port{
			ID:              portID,
			Name:            fmt.Sprintf("Analogue Output %d", i),
			ShortName:       fmt.Sprintf("Out%d", i),
			Type:            PortTypeAnalogue,
			Direction:       PortDirectionOutput,
			Number:          i,
			LevelMeterIndex: b.profile.GetLevelMeterIndex(portID),
		}

		// Level meter
		if port.LevelMeterIndex >= 0 {
			port.LevelMeterControl = b.findControl(b.profile.LevelMeterControlName(port.LevelMeterIndex))
		}

		device.Ports[portID] = port
		device.AnalogueOutputs = append(device.AnalogueOutputs, port)
	}

	return nil
}

func (b *DeviceBuilder) buildSPDIFInputs(device *Device) error {
	for i := 1; i <= b.profile.SPDIFInputCount(); i++ {
		portID := fmt.Sprintf("spdif-in-%d", i)

		port := &Port{
			ID:              portID,
			Name:            fmt.Sprintf("S/PDIF Input %d", i),
			ShortName:       fmt.Sprintf("SP%d", i),
			Type:            PortTypeSPDIF,
			Direction:       PortDirectionInput,
			Number:          i,
			LevelMeterIndex: b.profile.GetLevelMeterIndex(portID),
		}

		// Level meter
		if port.LevelMeterIndex >= 0 {
			port.LevelMeterControl = b.findControl(b.profile.LevelMeterControlName(port.LevelMeterIndex))
		}

		device.Ports[portID] = port
		device.SPDIFInputs = append(device.SPDIFInputs, port)
	}

	return nil
}

func (b *DeviceBuilder) buildSPDIFOutputs(device *Device) error {
	for i := 1; i <= b.profile.SPDIFOutputCount(); i++ {
		portID := fmt.Sprintf("spdif-out-%d", i)

		port := &Port{
			ID:              portID,
			Name:            fmt.Sprintf("S/PDIF Output %d", i),
			ShortName:       fmt.Sprintf("SPO%d", i),
			Type:            PortTypeSPDIF,
			Direction:       PortDirectionOutput,
			Number:          i,
			LevelMeterIndex: -1, // No level meters for S/PDIF outputs
		}

		device.Ports[portID] = port
		device.SPDIFOutputs = append(device.SPDIFOutputs, port)
	}

	return nil
}

func (b *DeviceBuilder) buildADATInputs(device *Device) error {
	for i := 1; i <= b.profile.ADATInputCount(); i++ {
		portID := fmt.Sprintf("adat-in-%d", i)

		port := &Port{
			ID:              portID,
			Name:            fmt.Sprintf("ADAT Input %d", i),
			ShortName:       fmt.Sprintf("AD%d", i),
			Type:            PortTypeADAT,
			Direction:       PortDirectionInput,
			Number:          i,
			LevelMeterIndex: b.profile.GetLevelMeterIndex(portID),
		}

		// Level meter
		if port.LevelMeterIndex >= 0 {
			port.LevelMeterControl = b.findControl(b.profile.LevelMeterControlName(port.LevelMeterIndex))
		}

		device.Ports[portID] = port
		device.ADATInputs = append(device.ADATInputs, port)
	}

	return nil
}

func (b *DeviceBuilder) buildADATOutputs(device *Device) error {
	for i := 1; i <= b.profile.ADATOutputCount(); i++ {
		portID := fmt.Sprintf("adat-out-%d", i)

		port := &Port{
			ID:              portID,
			Name:            fmt.Sprintf("ADAT Output %d", i),
			ShortName:       fmt.Sprintf("ADO%d", i),
			Type:            PortTypeADAT,
			Direction:       PortDirectionOutput,
			Number:          i,
			LevelMeterIndex: -1, // No level meters for ADAT outputs
		}

		device.Ports[portID] = port
		device.ADATOutputs = append(device.ADATOutputs, port)
	}

	return nil
}

func (b *DeviceBuilder) buildPCMPlayback(device *Device) error {
	for i := 1; i <= b.profile.PCMPlaybackCount(); i++ {
		portID := fmt.Sprintf("pcm-playback-%d", i)

		port := &Port{
			ID:              portID,
			Name:            fmt.Sprintf("PCM Playback %d", i),
			ShortName:       fmt.Sprintf("PCM%d", i),
			Type:            PortTypePCM,
			Direction:       PortDirectionInput, // From computer's perspective, playback goes INTO the interface
			Number:          i,
			LevelMeterIndex: b.profile.GetLevelMeterIndex(portID),
		}

		// Level meter
		if port.LevelMeterIndex >= 0 {
			port.LevelMeterControl = b.findControl(b.profile.LevelMeterControlName(port.LevelMeterIndex))
		}

		device.Ports[portID] = port
		device.PCMPlayback = append(device.PCMPlayback, port)
	}

	return nil
}

func (b *DeviceBuilder) buildPCMCapture(device *Device) error {
	for i := 1; i <= b.profile.PCMCaptureCount(); i++ {
		portID := fmt.Sprintf("pcm-capture-%d", i)

		port := &Port{
			ID:              portID,
			Name:            fmt.Sprintf("PCM Capture %d", i),
			ShortName:       fmt.Sprintf("CAP%d", i),
			Type:            PortTypePCM,
			Direction:       PortDirectionOutput, // From interface's perspective, capture goes OUT to computer
			Number:          i,
			LevelMeterIndex: -1, // No level meters for PCM capture
		}

		device.Ports[portID] = port
		device.PCMCapture = append(device.PCMCapture, port)
	}

	return nil
}

func (b *DeviceBuilder) buildMixes(device *Device) error {
	for i := 0; i < b.profile.MixCount(); i++ {
		letter := rune('A' + i)
		mixID := fmt.Sprintf("mix-%c", letter+'a'-'A') // lowercase: mix-a, mix-b, etc.

		mix := &Mix{
			ID:                    mixID,
			Name:                  fmt.Sprintf("Mix %c", letter),
			Letter:                letter,
			Index:                 i,
			OutputLevelMeterIndex: b.profile.GetLevelMeterIndex(mixID),
		}

		// Output level meter
		if mix.OutputLevelMeterIndex >= 0 {
			mix.OutputLevelMeterControl = b.findControl(b.profile.LevelMeterControlName(mix.OutputLevelMeterIndex))
		}

		// Build mix inputs
		for inputNum := 1; inputNum <= b.profile.MixInputCount(); inputNum++ {
			mixInput := &MixInput{
				Source:      device.MixInputSourcePort(inputNum),
				InputNumber: inputNum,
				Mix:         mix,
			}

			// Find volume control
			mixInput.VolumeControl = b.findControl(b.profile.MixInputVolumeControlName(letter, inputNum))

			mix.Inputs = append(mix.Inputs, mixInput)
		}

		device.Mixes = append(device.Mixes, mix)
		device.MixByID[mixID] = mix
	}

	return nil
}

func (b *DeviceBuilder) buildRoutingEndpoints(device *Device) error {
	// Analogue outputs
	for _, port := range device.AnalogueOutputs {
		ctl := b.findControl(b.profile.AnalogueOutputRoutingControlName(port.Number))
		if ctl == nil {
			continue // Skip if not found
		}

		endpoint := &RoutingEndpoint{
			Port:             port,
			RoutingControl:   ctl,
			AvailableSources: ctl.Items,
		}

		// Initialize current routing from hardware
		if err := endpoint.RefreshFromHardware(); err == nil {
			device.AnalogueOutputEndpoints = append(device.AnalogueOutputEndpoints, endpoint)
		}
	}

	// S/PDIF outputs
	for _, port := range device.SPDIFOutputs {
		ctl := b.findControl(b.profile.SPDIFOutputRoutingControlName(port.Number))
		if ctl == nil {
			continue
		}

		endpoint := &RoutingEndpoint{
			Port:             port,
			RoutingControl:   ctl,
			AvailableSources: ctl.Items,
		}

		if err := endpoint.RefreshFromHardware(); err == nil {
			device.SPDIFOutputEndpoints = append(device.SPDIFOutputEndpoints, endpoint)
		}
	}

	// ADAT outputs
	for _, port := range device.ADATOutputs {
		ctl := b.findControl(b.profile.ADATOutputRoutingControlName(port.Number))
		if ctl == nil {
			continue
		}

		endpoint := &RoutingEndpoint{
			Port:             port,
			RoutingControl:   ctl,
			AvailableSources: ctl.Items,
		}

		if err := endpoint.RefreshFromHardware(); err == nil {
			device.ADATOutputEndpoints = append(device.ADATOutputEndpoints, endpoint)
		}
	}

	// PCM capture
	for _, port := range device.PCMCapture {
		ctl := b.findControl(b.profile.PCMCaptureRoutingControlName(port.Number))
		if ctl == nil {
			continue
		}

		endpoint := &RoutingEndpoint{
			Port:             port,
			RoutingControl:   ctl,
			AvailableSources: ctl.Items,
		}

		if err := endpoint.RefreshFromHardware(); err == nil {
			device.PCMCaptureEndpoints = append(device.PCMCaptureEndpoints, endpoint)
		}
	}

	return nil
}

func (b *DeviceBuilder) buildMixerInputEndpoints(device *Device) error {
	for inputNum := 1; inputNum <= b.profile.MixInputCount(); inputNum++ {
		ctl := b.findControl(b.profile.MixerInputRoutingControlName(inputNum))
		if ctl == nil {
			continue
		}

		// Create a port for the mixer input slot
		port := &Port{
			ID:              fmt.Sprintf("mixer-input-%d", inputNum),
			Name:            fmt.Sprintf("Mixer %d", inputNum),
			ShortName:       fmt.Sprintf("Mix%d", inputNum),
			Type:            PortTypeMixer,
			Direction:       PortDirectionInput,
			Number:          inputNum,
			LevelMeterIndex: -1,
		}

		endpoint := &RoutingEndpoint{
			Port:             port,
			RoutingControl:   ctl,
			AvailableSources: ctl.Items,
		}

		if err := endpoint.RefreshFromHardware(); err == nil {
			device.MixerInputEndpoints = append(device.MixerInputEndpoints, endpoint)
		}
	}

	return nil
}

func (b *DeviceBuilder) collectLevelMeters(device *Device) {
	// Collect all level meter controls in order
	device.LevelMeterControls = make([]*scarlettctl.Control, b.profile.LevelMeterCount())

	for i := 0; i < b.profile.LevelMeterCount(); i++ {
		meterName := b.profile.LevelMeterControlName(i)
		if ctl := b.findControl(meterName); ctl != nil {
			device.LevelMeterControls[i] = ctl
		}
	}
}

// buildControlIndex fetches all controls once and builds lookup maps
func (b *DeviceBuilder) buildControlIndex() error {
	controls, err := b.card.GetControls()
	if err != nil {
		return err
	}

	b.controlsByName = make(map[string]*scarlettctl.Control, len(controls))
	b.controlsByID = make(map[string]*scarlettctl.Control, len(controls))

	for _, ctl := range controls {
		b.controlsByName[ctl.Name] = ctl
		b.controlsByID[ctl.FullID()] = ctl
	}

	return nil
}

// findControl looks up a control by name or ID using the pre-built index
func (b *DeviceBuilder) findControl(name string) *scarlettctl.Control {
	// Try full ID lookup if input looks like an ID
	if strings.Contains(name, ":") && strings.Contains(name, "/") {
		if ctl, ok := b.controlsByID[name]; ok {
			return ctl
		}
		return nil
	}

	// Otherwise look up by name
	if ctl, ok := b.controlsByName[name]; ok {
		return ctl
	}
	return nil
}
