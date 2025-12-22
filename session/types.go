package session

// SessionConfig is the root configuration for sessionmixer.
// It defines devices (named audio sources) and cue mixes.
type SessionConfig struct {
	// Card is the ALSA card number for the Scarlett interface
	Card int `yaml:"card" dd:"+required"`

	// LevelSmoothing is the number of samples to average for level meters (0 = disabled)
	LevelSmoothing int `yaml:"level_smoothing"`

	// Devices are named audio sources that map to topology ports
	Devices []DeviceConfig `yaml:"devices"`

	// CueMixes are stereo mix allocations routed to outputs
	CueMixes []CueMixConfig `yaml:"cue_mixes"`
}

// DeviceConfig represents a named audio device in the session.
// A device maps to one or two topology ports (mono or stereo).
type DeviceConfig struct {
	// Name is the display name for this device (e.g., "Guitar FX", "Vocal Mic")
	Name string `yaml:"name" dd:"+required"`

	// Ports are the topology port IDs this device uses.
	// One port for mono, two ports for stereo.
	// Examples: ["analogue-in-1"], ["pcm-playback-1", "pcm-playback-2"]
	Ports []string `yaml:"ports" dd:"+required"`
}

// IsStereo returns true if this device has two ports (stereo)
func (d *DeviceConfig) IsStereo() bool {
	return len(d.Ports) == 2
}

// CueMixConfig represents a cue mix allocation.
// A cue mix uses one or two mixes (mono or stereo) and routes to hardware outputs.
type CueMixConfig struct {
	// Name is the display name for this cue mix (e.g., "Drummer Cue")
	Name string `yaml:"name" dd:"+required"`

	// MixPair specifies which mixes to use.
	// For stereo: "A+B", "C+D", etc.
	// For mono: "A", "C", etc.
	MixPair string `yaml:"mix_pair" dd:"+required"`

	// Outputs are the hardware output port IDs to route to.
	// One port for mono, two for stereo.
	// Examples: ["analogue-out-1", "analogue-out-2"]
	Outputs []string `yaml:"outputs" dd:"+required"`

	// Devices lists the device names to include in this cue mix.
	// Each device gets a fader controlling its level in the mix.
	Devices []string `yaml:"devices"`
}

// IsStereo returns true if this cue mix uses a stereo pair (e.g., "A+B")
func (c *CueMixConfig) IsStereo() bool {
	// Stereo pairs contain a '+' character
	for _, ch := range c.MixPair {
		if ch == '+' {
			return true
		}
	}
	return false
}

// ParseMixPair extracts the mix letters from the MixPair string.
// Returns (leftLetter, rightLetter, isStereo).
// For "A+B" returns ('A', 'B', true).
// For "A" returns ('A', 0, false).
func (c *CueMixConfig) ParseMixPair() (left rune, right rune, stereo bool) {
	if len(c.MixPair) == 0 {
		return 0, 0, false
	}

	// Check for stereo format "X+Y"
	for i, ch := range c.MixPair {
		if ch == '+' && i > 0 && i < len(c.MixPair)-1 {
			runes := []rune(c.MixPair)
			return runes[0], runes[i+1], true
		}
	}

	// Mono format - just a single letter
	return rune(c.MixPair[0]), 0, false
}
