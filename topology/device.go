package topology

import "github.com/michaelquigley/scarlettctl"

// Device represents the complete topology of an audio interface.
// It contains all ports, mixes, and routing endpoints discovered from hardware.
type Device struct {
	// Device identification
	Name string
	Card *scarlettctl.Card

	// All ports indexed by ID
	Ports map[string]*Port

	// Categorized port slices for easy iteration
	AnalogueInputs  []*Port
	AnalogueOutputs []*Port
	SPDIFInputs     []*Port
	SPDIFOutputs    []*Port
	ADATInputs      []*Port
	ADATOutputs     []*Port
	PCMPlayback     []*Port // From computer to interface
	PCMCapture      []*Port // From interface to computer

	// Mixes (A-L on 18i20)
	Mixes   []*Mix
	MixByID map[string]*Mix

	// Routable endpoints (outputs and PCM capture that can have sources routed to them)
	AnalogueOutputEndpoints []*RoutingEndpoint
	SPDIFOutputEndpoints    []*RoutingEndpoint
	ADATOutputEndpoints     []*RoutingEndpoint
	PCMCaptureEndpoints     []*RoutingEndpoint
	MixerInputEndpoints     []*RoutingEndpoint

	// All level meters for bulk reading
	LevelMeterControls []*scarlettctl.Control

	// Profile used to build this device
	Profile DeviceProfile
}

// GetPort returns a port by ID, or nil if not found
func (d *Device) GetPort(id string) *Port {
	return d.Ports[id]
}

// GetMix returns a mix by ID, or nil if not found
func (d *Device) GetMix(id string) *Mix {
	return d.MixByID[id]
}

// GetMixByLetter returns a mix by its letter (A-L), or nil if not found
func (d *Device) GetMixByLetter(letter rune) *Mix {
	idx := int(letter - 'A')
	if idx >= 0 && idx < len(d.Mixes) {
		return d.Mixes[idx]
	}
	// Also try lowercase
	idx = int(letter - 'a')
	if idx >= 0 && idx < len(d.Mixes) {
		return d.Mixes[idx]
	}
	return nil
}

// GetMixByIndex returns a mix by its 0-based index, or nil if out of range
func (d *Device) GetMixByIndex(index int) *Mix {
	if index >= 0 && index < len(d.Mixes) {
		return d.Mixes[index]
	}
	return nil
}

// AllInputPorts returns all input ports (hardware + PCM playback)
func (d *Device) AllInputPorts() []*Port {
	var ports []*Port
	ports = append(ports, d.AnalogueInputs...)
	ports = append(ports, d.SPDIFInputs...)
	ports = append(ports, d.ADATInputs...)
	ports = append(ports, d.PCMPlayback...)
	return ports
}

// AllOutputPorts returns all output ports (hardware + PCM capture)
func (d *Device) AllOutputPorts() []*Port {
	var ports []*Port
	ports = append(ports, d.AnalogueOutputs...)
	ports = append(ports, d.SPDIFOutputs...)
	ports = append(ports, d.ADATOutputs...)
	ports = append(ports, d.PCMCapture...)
	return ports
}

// AllHardwareInputs returns all physical hardware inputs (not PCM)
func (d *Device) AllHardwareInputs() []*Port {
	var ports []*Port
	ports = append(ports, d.AnalogueInputs...)
	ports = append(ports, d.SPDIFInputs...)
	ports = append(ports, d.ADATInputs...)
	return ports
}

// AllHardwareOutputs returns all physical hardware outputs (not PCM)
func (d *Device) AllHardwareOutputs() []*Port {
	var ports []*Port
	ports = append(ports, d.AnalogueOutputs...)
	ports = append(ports, d.SPDIFOutputs...)
	ports = append(ports, d.ADATOutputs...)
	return ports
}

// AllRoutingEndpoints returns all routing endpoints
func (d *Device) AllRoutingEndpoints() []*RoutingEndpoint {
	var endpoints []*RoutingEndpoint
	endpoints = append(endpoints, d.AnalogueOutputEndpoints...)
	endpoints = append(endpoints, d.SPDIFOutputEndpoints...)
	endpoints = append(endpoints, d.ADATOutputEndpoints...)
	endpoints = append(endpoints, d.PCMCaptureEndpoints...)
	endpoints = append(endpoints, d.MixerInputEndpoints...)
	return endpoints
}

// GetRoutingEndpointForPort returns the routing endpoint for a given port, or nil
func (d *Device) GetRoutingEndpointForPort(port *Port) *RoutingEndpoint {
	for _, ep := range d.AllRoutingEndpoints() {
		if ep.Port == port {
			return ep
		}
	}
	return nil
}

// MixInputSourcePort returns the source port for a given mix input number.
// For 18i20: 1-24 = PCM, 25-33 = Analogue In, 34-35 = S/PDIF, 36-43 = ADAT
func (d *Device) MixInputSourcePort(inputNum int) *Port {
	switch {
	case inputNum >= 1 && inputNum <= d.Profile.PCMPlaybackCount():
		idx := inputNum - 1
		if idx < len(d.PCMPlayback) {
			return d.PCMPlayback[idx]
		}
	case inputNum <= d.Profile.PCMPlaybackCount()+d.Profile.AnalogueInputCount():
		idx := inputNum - d.Profile.PCMPlaybackCount() - 1
		if idx < len(d.AnalogueInputs) {
			return d.AnalogueInputs[idx]
		}
	case inputNum <= d.Profile.PCMPlaybackCount()+d.Profile.AnalogueInputCount()+d.Profile.SPDIFInputCount():
		idx := inputNum - d.Profile.PCMPlaybackCount() - d.Profile.AnalogueInputCount() - 1
		if idx < len(d.SPDIFInputs) {
			return d.SPDIFInputs[idx]
		}
	default:
		idx := inputNum - d.Profile.PCMPlaybackCount() - d.Profile.AnalogueInputCount() - d.Profile.SPDIFInputCount() - 1
		if idx >= 0 && idx < len(d.ADATInputs) {
			return d.ADATInputs[idx]
		}
	}
	return nil
}

// PortCount returns the total number of ports
func (d *Device) PortCount() int {
	return len(d.Ports)
}

// MixCount returns the number of mixes
func (d *Device) MixCount() int {
	return len(d.Mixes)
}

// String returns a summary of the device
func (d *Device) String() string {
	return d.Name
}
