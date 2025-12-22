package topology

import (
	"fmt"

	"github.com/michaelquigley/scarlettctl"
)

// FirmwareVersion represents a 4-part firmware version read from hardware.
// The 4 indexed controls map to Major.Minor.Patch.Build.
type FirmwareVersion struct {
	Major uint16 // Index 0
	Minor uint16 // Index 1
	Patch uint16 // Index 2
	Build uint16 // Index 3
}

// String returns the firmware version in "Major.Minor.Patch.Build" format.
func (v FirmwareVersion) String() string {
	return fmt.Sprintf("%d.%d.%d.%d", v.Major, v.Minor, v.Patch, v.Build)
}

// Equal returns true if two versions are exactly equal.
func (v FirmwareVersion) Equal(other FirmwareVersion) bool {
	return v.Major == other.Major &&
		v.Minor == other.Minor &&
		v.Patch == other.Patch &&
		v.Build == other.Build
}

// IsZero returns true if all components are zero (uninitialized/unknown).
func (v FirmwareVersion) IsZero() bool {
	return v.Major == 0 && v.Minor == 0 && v.Patch == 0 && v.Build == 0
}

// FirmwareInfo contains both app and ESP firmware versions.
type FirmwareInfo struct {
	App FirmwareVersion
	ESP FirmwareVersion
}

// ReadFirmwareVersion reads a 4-part firmware version from indexed controls.
// controlBaseName is "Firmware Version" or "ESP Firmware Version".
func ReadFirmwareVersion(card *scarlettctl.Card, controlBaseName string) (FirmwareVersion, error) {
	controls, err := card.GetControls()
	if err != nil {
		return FirmwareVersion{}, fmt.Errorf("getting controls: %w", err)
	}

	// Collect indexed controls with matching name
	indexedValues := make(map[int]uint16)
	for _, ctl := range controls {
		if ctl.Name == controlBaseName {
			val, err := ctl.GetValue()
			if err != nil {
				return FirmwareVersion{}, fmt.Errorf("reading %s[%d]: %w",
					controlBaseName, ctl.Index, err)
			}
			indexedValues[ctl.Index] = uint16(val)
		}
	}

	// Verify we found all 4 indices
	if len(indexedValues) != 4 {
		return FirmwareVersion{}, fmt.Errorf("expected 4 indexed controls for %s, found %d",
			controlBaseName, len(indexedValues))
	}

	return FirmwareVersion{
		Major: indexedValues[0],
		Minor: indexedValues[1],
		Patch: indexedValues[2],
		Build: indexedValues[3],
	}, nil
}

// ReadFirmwareInfo reads both app and ESP firmware versions from the card.
func ReadFirmwareInfo(card *scarlettctl.Card) (FirmwareInfo, error) {
	app, err := ReadFirmwareVersion(card, "Firmware Version")
	if err != nil {
		return FirmwareInfo{}, fmt.Errorf("reading app firmware: %w", err)
	}

	esp, err := ReadFirmwareVersion(card, "ESP Firmware Version")
	if err != nil {
		return FirmwareInfo{}, fmt.Errorf("reading ESP firmware: %w", err)
	}

	return FirmwareInfo{App: app, ESP: esp}, nil
}
