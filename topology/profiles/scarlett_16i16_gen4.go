package profiles

import (
	"fmt"

	"github.com/michaelquigley/sessionmixer/topology"
)

func init() {
	RegisterProfile(topology.ProfileEntry{
		CardNameContains: "16i16",
		AppFirmware:      topology.FirmwareVersion{Major: 2, Minor: 0, Patch: 2464, Build: 0},
		ESPFirmware:      topology.FirmwareVersion{Major: 1, Minor: 0, Patch: 0, Build: 364},
		CreateProfile: func(app, esp topology.FirmwareVersion) topology.DeviceProfile {
			return NewScarlett16i16Gen4Profile(app, esp)
		},
	})
}

// Scarlett16i16Gen4Profile implements DeviceProfile for the 16i16 4th Gen
type Scarlett16i16Gen4Profile struct {
	levelMeterMap map[string]int
	// Expected firmware versions
	appFirmware topology.FirmwareVersion
	espFirmware topology.FirmwareVersion
}

// NewScarlett16i16Gen4Profile creates a new profile for the 16i16 Gen 4
func NewScarlett16i16Gen4Profile(appFirmware, espFirmware topology.FirmwareVersion) *Scarlett16i16Gen4Profile {
	p := &Scarlett16i16Gen4Profile{
		levelMeterMap: make(map[string]int),
		appFirmware:   appFirmware,
		espFirmware:   espFirmware,
	}
	p.buildLevelMeterMap()
	return p
}

// buildLevelMeterMap populates the level meter index lookup table
// Based on docs/16i16g4-layout.md
func (p *Scarlett16i16Gen4Profile) buildLevelMeterMap() {
	// Analogue Inputs 1-6 -> meters [0-5]
	for i := 1; i <= 6; i++ {
		p.levelMeterMap[fmt.Sprintf("analogue-in-%d", i)] = i - 1
	}

	// S/PDIF Inputs 1-2 -> meters [6-7]
	for i := 1; i <= 2; i++ {
		p.levelMeterMap[fmt.Sprintf("spdif-in-%d", i)] = 5 + i
	}

	// ADAT Inputs 1-8 -> meters [8-15]
	for i := 1; i <= 8; i++ {
		p.levelMeterMap[fmt.Sprintf("adat-in-%d", i)] = 7 + i
	}

	// PCM Playback 1-18 -> meters [16-33]
	for i := 1; i <= 18; i++ {
		p.levelMeterMap[fmt.Sprintf("pcm-playback-%d", i)] = 15 + i
	}

	// Mix outputs 1-12 (A-L) -> meters [34-45]
	for i := 0; i < 12; i++ {
		p.levelMeterMap[fmt.Sprintf("mix-%c", 'a'+i)] = 34 + i
	}

	// Analogue Outputs 1-8 -> meters [46-53]
	for i := 1; i <= 8; i++ {
		p.levelMeterMap[fmt.Sprintf("analogue-out-%d", i)] = 45 + i
	}
}

func (p *Scarlett16i16Gen4Profile) Name() string              { return "Scarlett 16i16 4th Gen" }
func (p *Scarlett16i16Gen4Profile) Generation() int           { return 4 }
func (p *Scarlett16i16Gen4Profile) AnalogueInputCount() int   { return 6 }
func (p *Scarlett16i16Gen4Profile) AnalogueOutputCount() int  { return 8 }
func (p *Scarlett16i16Gen4Profile) SPDIFInputCount() int      { return 2 }
func (p *Scarlett16i16Gen4Profile) SPDIFOutputCount() int     { return 2 }
func (p *Scarlett16i16Gen4Profile) ADATInputCount() int       { return 8 }
func (p *Scarlett16i16Gen4Profile) ADATOutputCount() int      { return 8 }
func (p *Scarlett16i16Gen4Profile) PCMPlaybackCount() int     { return 18 }
func (p *Scarlett16i16Gen4Profile) PCMCaptureCount() int      { return 18 }
func (p *Scarlett16i16Gen4Profile) MixCount() int             { return 12 }
func (p *Scarlett16i16Gen4Profile) MixInputCount() int        { return 34 }
func (p *Scarlett16i16Gen4Profile) LevelMeterCount() int      { return 54 }

func (p *Scarlett16i16Gen4Profile) GetLevelMeterIndex(portID string) int {
	if idx, ok := p.levelMeterMap[portID]; ok {
		return idx
	}
	return -1
}

// Control name patterns for Gen 4 (same format as 18i20)

func (p *Scarlett16i16Gen4Profile) MixInputVolumeControlName(mixLetter rune, inputNum int) string {
	return fmt.Sprintf("Mix %c Input %02d Playback Volume", mixLetter, inputNum)
}

func (p *Scarlett16i16Gen4Profile) AnalogueOutputRoutingControlName(portNum int) string {
	return fmt.Sprintf("Analogue Output %02d Playback Enum", portNum)
}

func (p *Scarlett16i16Gen4Profile) SPDIFOutputRoutingControlName(portNum int) string {
	return fmt.Sprintf("S/PDIF Output %02d Playback Enum", portNum)
}

func (p *Scarlett16i16Gen4Profile) ADATOutputRoutingControlName(portNum int) string {
	return fmt.Sprintf("ADAT Output %02d Playback Enum", portNum)
}

func (p *Scarlett16i16Gen4Profile) PCMCaptureRoutingControlName(portNum int) string {
	return fmt.Sprintf("PCM %02d Capture Enum", portNum)
}

func (p *Scarlett16i16Gen4Profile) LevelMeterControlName(index int) string {
	return fmt.Sprintf("pcm:0.0/Level Meter[%d]", index)
}

// Input capabilities - based on 16i16 Gen 4 hardware
// Inputs 1-2 are combo XLR/TRS with phantom, air, pad, gain
// Inputs 3-6 are line inputs (no preamp features)

func (p *Scarlett16i16Gen4Profile) HasPhantom(portNum int) bool {
	return portNum >= 1 && portNum <= 2
}

func (p *Scarlett16i16Gen4Profile) HasAir(portNum int) bool {
	return portNum >= 1 && portNum <= 2
}

func (p *Scarlett16i16Gen4Profile) HasPad(portNum int) bool {
	return portNum >= 1 && portNum <= 2
}

func (p *Scarlett16i16Gen4Profile) HasGain(portNum int) bool {
	return portNum >= 1 && portNum <= 2
}

func (p *Scarlett16i16Gen4Profile) GainControlName(portNum int) string {
	return fmt.Sprintf("Analogue %d Gain", portNum)
}

func (p *Scarlett16i16Gen4Profile) PhantomControlName(portNum int) string {
	return fmt.Sprintf("Analogue %d Phantom Power", portNum)
}

func (p *Scarlett16i16Gen4Profile) AirControlName(portNum int) string {
	return fmt.Sprintf("Analogue %d Air", portNum)
}

func (p *Scarlett16i16Gen4Profile) PadControlName(portNum int) string {
	return fmt.Sprintf("Analogue %d Pad", portNum)
}

func (p *Scarlett16i16Gen4Profile) ExpectedAppFirmware() topology.FirmwareVersion {
	return p.appFirmware
}

func (p *Scarlett16i16Gen4Profile) ExpectedESPFirmware() topology.FirmwareVersion {
	return p.espFirmware
}

func (p *Scarlett16i16Gen4Profile) IsCompatibleWith(info topology.FirmwareInfo) bool {
	return p.appFirmware.Equal(info.App) && p.espFirmware.Equal(info.ESP)
}

// Ensure Scarlett16i16Gen4Profile implements DeviceProfile
var _ topology.DeviceProfile = (*Scarlett16i16Gen4Profile)(nil)
