package topology

import (
	"sync/atomic"

	"github.com/michaelquigley/scarlettctl"
)

// PortType identifies the physical connection type
type PortType string

const (
	PortTypeAnalogue PortType = "Analogue"
	PortTypeSPDIF    PortType = "S/PDIF"
	PortTypeADAT     PortType = "ADAT"
	PortTypePCM      PortType = "PCM"
)

// PortDirection indicates whether a port is an input or output
type PortDirection string

const (
	PortDirectionInput  PortDirection = "input"
	PortDirectionOutput PortDirection = "output"
)

// Port represents a single mono audio channel (hardware or virtual).
// This is the fundamental building block - all inputs and outputs are Ports.
type Port struct {
	// Identity
	ID        string // Unique identifier e.g., "analogue-in-1", "pcm-out-5"
	Name      string // Human-readable name e.g., "Analogue Input 1"
	ShortName string // Abbreviated name e.g., "An1", "PCM5"

	// Classification
	Type      PortType      // Analogue, S/PDIF, ADAT, PCM
	Direction PortDirection // Input or Output
	Number    int           // 1-based port number within its type

	// Capabilities (for hardware inputs)
	HasPhantom bool // Can enable phantom power
	HasAir     bool // Air mode available
	HasPad     bool // Has -10dB pad
	HasGain    bool // Has adjustable preamp gain

	// Associated ALSA controls (may be nil if not applicable)
	LevelMeterControl *scarlettctl.Control // Level meter (read-only)
	GainControl       *scarlettctl.Control // Preamp gain
	PhantomControl    *scarlettctl.Control // Phantom power switch
	AirControl        *scarlettctl.Control // Air mode switch
	PadControl        *scarlettctl.Control // Pad switch

	// Level meter index for direct lookup (-1 if none)
	LevelMeterIndex int
}

// Mix represents one mixer bus (A-L on the 18i20g4).
// Each mix has multiple inputs, each with its own volume control.
type Mix struct {
	// Identity
	ID     string // "mix-a", "mix-b", etc.
	Name   string // "Mix A", "Mix B", etc.
	Letter rune   // 'A', 'B', etc.
	Index  int    // 0-based index (A=0, B=1, etc.)

	// Inputs - each input has a volume control
	Inputs []*MixInput

	// Output level meter
	OutputLevelMeterControl *scarlettctl.Control
	OutputLevelMeterIndex   int

	// Gang state (for stereo linking) - managed by DeviceState
	// This is here for reference but actual gang state is in DeviceState
}

// MixInput represents one input to a mix bus with its volume control
type MixInput struct {
	// Source port (may be nil if not yet resolved)
	Source *Port

	// Input number within the mix (1-based, e.g., 1-43 for 18i20g4)
	InputNumber int

	// Volume control for this input
	VolumeControl *scarlettctl.Control

	// Parent mix reference
	Mix *Mix
}

// RoutingEndpoint represents a destination that can have a source routed to it.
// Examples: Hardware outputs, PCM capture channels
type RoutingEndpoint struct {
	// The port this endpoint corresponds to
	Port *Port

	// Routing control (enumerated type listing available sources)
	RoutingControl *scarlettctl.Control

	// Available sources (from the enum items)
	AvailableSources []string

	// Current routing state (updated atomically)
	currentSourceIndex int64
}

// GetCurrentSourceIndex returns the current source index atomically
func (re *RoutingEndpoint) GetCurrentSourceIndex() int {
	return int(atomic.LoadInt64(&re.currentSourceIndex))
}

// GetCurrentSource returns the name of the currently routed source
func (re *RoutingEndpoint) GetCurrentSource() string {
	idx := re.GetCurrentSourceIndex()
	if idx >= 0 && idx < len(re.AvailableSources) {
		return re.AvailableSources[idx]
	}
	return "Unknown"
}

// SetSourceIndex atomically updates the current source (called from event monitor)
func (re *RoutingEndpoint) SetSourceIndex(idx int) {
	atomic.StoreInt64(&re.currentSourceIndex, int64(idx))
}

// RefreshFromHardware reads the current routing value from hardware
func (re *RoutingEndpoint) RefreshFromHardware() error {
	if re.RoutingControl == nil {
		return nil
	}
	val, err := re.RoutingControl.GetValue()
	if err != nil {
		return err
	}
	re.SetSourceIndex(int(val))
	return nil
}
