package topology

// DeviceProfile defines the capabilities of a specific Focusrite interface model.
// This enables supporting different devices (18i20, 16i16, 4i4, etc.)
//
// Profile implementations are in the profiles subpackage and self-register
// via init() functions.
type DeviceProfile interface {
	// Identity
	Name() string
	Generation() int

	// Port counts
	AnalogueInputCount() int
	AnalogueOutputCount() int
	SPDIFInputCount() int
	SPDIFOutputCount() int
	ADATInputCount() int
	ADATOutputCount() int
	PCMPlaybackCount() int
	PCMCaptureCount() int

	// Mixing
	MixCount() int
	MixInputCount() int

	// Level meters
	LevelMeterCount() int
	GetLevelMeterIndex(portID string) int

	// Control name patterns (vary by generation)
	MixInputVolumeControlName(mixLetter rune, inputNum int) string
	AnalogueOutputRoutingControlName(portNum int) string
	SPDIFOutputRoutingControlName(portNum int) string
	ADATOutputRoutingControlName(portNum int) string
	PCMCaptureRoutingControlName(portNum int) string
	MixerInputRoutingControlName(inputNum int) string
	LevelMeterControlName(index int) string

	// Input capabilities by port number
	HasPhantom(portNum int) bool
	HasAir(portNum int) bool
	HasPad(portNum int) bool
	HasGain(portNum int) bool

	// Preamp control names
	GainControlName(portNum int) string
	PhantomControlName(portNum int) string
	AirControlName(portNum int) string
	PadControlName(portNum int) string

	// Firmware compatibility
	ExpectedAppFirmware() FirmwareVersion
	ExpectedESPFirmware() FirmwareVersion
	IsCompatibleWith(info FirmwareInfo) bool
}
