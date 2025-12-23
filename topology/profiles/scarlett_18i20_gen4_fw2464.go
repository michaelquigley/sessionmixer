package profiles

import (
	"fmt"

	"github.com/michaelquigley/sessionmixer/topology"
)

func init() {
	RegisterProfile(topology.ProfileEntry{
		CardNameContains: "18i20",
		AppFirmware:      topology.FirmwareVersion{Major: 2, Minor: 0, Patch: 2464, Build: 0},
		ESPFirmware:      topology.FirmwareVersion{Major: 1, Minor: 0, Patch: 0, Build: 364},
		CreateProfile: func(app, esp topology.FirmwareVersion) topology.DeviceProfile {
			return NewScarlett18i20Gen4FW2464Profile(app, esp)
		},
	})
}

// Scarlett18i20Gen4FW2464Profile implements DeviceProfile for the 18i20 4th Gen with firmware 2464
type Scarlett18i20Gen4FW2464Profile struct {
	// Level meter index lookup table
	levelMeterMap map[string]int
	// Expected firmware versions
	appFirmware topology.FirmwareVersion
	espFirmware topology.FirmwareVersion
}

// NewScarlett18i20Gen4FW2464Profile creates a new profile for the 18i20 Gen 4 (FW 2464)
func NewScarlett18i20Gen4FW2464Profile(appFirmware, espFirmware topology.FirmwareVersion) *Scarlett18i20Gen4FW2464Profile {
	p := &Scarlett18i20Gen4FW2464Profile{
		levelMeterMap: make(map[string]int),
		appFirmware:   appFirmware,
		espFirmware:   espFirmware,
	}
	p.buildLevelMeterMap()
	return p
}

// buildLevelMeterMap populates the level meter index lookup table
// Based on docs/18i20g4-2464-layout.md
func (p *Scarlett18i20Gen4FW2464Profile) buildLevelMeterMap() {
	// Analogue Inputs 1-9 -> meters [0-8]
	for i := 1; i <= 9; i++ {
		p.levelMeterMap[fmt.Sprintf("analogue-in-%d", i)] = i - 1
	}

	// S/PDIF Inputs 1-2 -> meters [9-10]
	for i := 1; i <= 2; i++ {
		p.levelMeterMap[fmt.Sprintf("spdif-in-%d", i)] = 8 + i
	}

	// ADAT Inputs 1-16 -> meters [11-26]
	for i := 1; i <= 16; i++ {
		p.levelMeterMap[fmt.Sprintf("adat-in-%d", i)] = 10 + i
	}

	// PCM Playback 1-24 -> meters [27-50]
	for i := 1; i <= 24; i++ {
		p.levelMeterMap[fmt.Sprintf("pcm-playback-%d", i)] = 26 + i
	}

	// Mix outputs 1-12 (A-L) -> meters [53-64]
	for i := 0; i < 12; i++ {
		p.levelMeterMap[fmt.Sprintf("mix-%c", 'a'+i)] = 53 + i
	}

	// Analogue Outputs 1-4 -> meters [65-68]
	// Note: FW 2464 only has meters for analogue outputs 1-4 (not 1-14 like FW 2399)
	for i := 1; i <= 4; i++ {
		p.levelMeterMap[fmt.Sprintf("analogue-out-%d", i)] = 64 + i
	}

	// Note: Meters [51-52] are unused/undefined in the documentation
	// S/PDIF and ADAT outputs don't have level meters in the layout
	// PCM Capture channels don't have level meters (they're virtual)
}

func (p *Scarlett18i20Gen4FW2464Profile) Name() string             { return "Scarlett 18i20 4th Gen (FW 2464)" }
func (p *Scarlett18i20Gen4FW2464Profile) Generation() int          { return 4 }
func (p *Scarlett18i20Gen4FW2464Profile) AnalogueInputCount() int  { return 9 }
func (p *Scarlett18i20Gen4FW2464Profile) AnalogueOutputCount() int { return 14 }
func (p *Scarlett18i20Gen4FW2464Profile) SPDIFInputCount() int     { return 2 }
func (p *Scarlett18i20Gen4FW2464Profile) SPDIFOutputCount() int    { return 2 }
func (p *Scarlett18i20Gen4FW2464Profile) ADATInputCount() int      { return 16 }
func (p *Scarlett18i20Gen4FW2464Profile) ADATOutputCount() int     { return 16 }
func (p *Scarlett18i20Gen4FW2464Profile) PCMPlaybackCount() int    { return 24 }
func (p *Scarlett18i20Gen4FW2464Profile) PCMCaptureCount() int     { return 26 }
func (p *Scarlett18i20Gen4FW2464Profile) MixCount() int            { return 12 }
func (p *Scarlett18i20Gen4FW2464Profile) MixInputCount() int       { return 53 }
func (p *Scarlett18i20Gen4FW2464Profile) LevelMeterCount() int     { return 69 }

func (p *Scarlett18i20Gen4FW2464Profile) GetLevelMeterIndex(portID string) int {
	if idx, ok := p.levelMeterMap[portID]; ok {
		return idx
	}
	return -1
}

// Control name patterns for Gen 4

func (p *Scarlett18i20Gen4FW2464Profile) MixInputVolumeControlName(mixLetter rune, inputNum int) string {
	return fmt.Sprintf("Mix %c Input %02d Playback Volume", mixLetter, inputNum)
}

func (p *Scarlett18i20Gen4FW2464Profile) AnalogueOutputRoutingControlName(portNum int) string {
	return fmt.Sprintf("Analogue %d Playback Enum", portNum)
}

func (p *Scarlett18i20Gen4FW2464Profile) SPDIFOutputRoutingControlName(portNum int) string {
	return fmt.Sprintf("S/PDIF %d Playback Enum", portNum)
}

func (p *Scarlett18i20Gen4FW2464Profile) ADATOutputRoutingControlName(portNum int) string {
	return fmt.Sprintf("ADAT %d Playback Enum", portNum)
}

func (p *Scarlett18i20Gen4FW2464Profile) PCMCaptureRoutingControlName(portNum int) string {
	return fmt.Sprintf("PCM %d Capture Enum", portNum)
}

func (p *Scarlett18i20Gen4FW2464Profile) MixerInputRoutingControlName(inputNum int) string {
	return fmt.Sprintf("Mixer %d Capture Enum", inputNum)
}

func (p *Scarlett18i20Gen4FW2464Profile) LevelMeterControlName(index int) string {
	return fmt.Sprintf("pcm:0.0/Level Meter[%d]", index)
}

// Input capabilities - based on 18i20 Gen 4 hardware
// Inputs 1-2 are combo XLR/TRS with phantom, air, pad
// Inputs 3-4 are combo XLR/TRS with phantom (no air/pad in Gen 4)
// Inputs 5-8 are line inputs
// Input 9 is also a line input (rear)

func (p *Scarlett18i20Gen4FW2464Profile) HasPhantom(portNum int) bool {
	return portNum >= 1 && portNum <= 4
}

func (p *Scarlett18i20Gen4FW2464Profile) HasAir(portNum int) bool {
	return portNum >= 1 && portNum <= 2
}

func (p *Scarlett18i20Gen4FW2464Profile) HasPad(portNum int) bool {
	return portNum >= 1 && portNum <= 2
}

func (p *Scarlett18i20Gen4FW2464Profile) HasGain(portNum int) bool {
	return portNum >= 1 && portNum <= 8 // Inputs 1-8 have gain control
}

func (p *Scarlett18i20Gen4FW2464Profile) GainControlName(portNum int) string {
	return fmt.Sprintf("Analogue %d Gain", portNum)
}

func (p *Scarlett18i20Gen4FW2464Profile) PhantomControlName(portNum int) string {
	return fmt.Sprintf("Analogue %d Phantom Power", portNum)
}

func (p *Scarlett18i20Gen4FW2464Profile) AirControlName(portNum int) string {
	return fmt.Sprintf("Analogue %d Air", portNum)
}

func (p *Scarlett18i20Gen4FW2464Profile) PadControlName(portNum int) string {
	return fmt.Sprintf("Analogue %d Pad", portNum)
}

func (p *Scarlett18i20Gen4FW2464Profile) ExpectedAppFirmware() topology.FirmwareVersion {
	return p.appFirmware
}

func (p *Scarlett18i20Gen4FW2464Profile) ExpectedESPFirmware() topology.FirmwareVersion {
	return p.espFirmware
}

func (p *Scarlett18i20Gen4FW2464Profile) IsCompatibleWith(info topology.FirmwareInfo) bool {
	return p.appFirmware.Equal(info.App) && p.espFirmware.Equal(info.ESP)
}

// Ensure Scarlett18i20Gen4FW2464Profile implements DeviceProfile
var _ topology.DeviceProfile = (*Scarlett18i20Gen4FW2464Profile)(nil)
