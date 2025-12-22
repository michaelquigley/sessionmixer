package topology

import (
	"fmt"
	"strings"

	"github.com/michaelquigley/scarlettctl"
)

// ProfileEntry defines a profile with its expected firmware versions.
type ProfileEntry struct {
	CardNameContains string
	AppFirmware      FirmwareVersion
	ESPFirmware      FirmwareVersion
	CreateProfile    func(app, esp FirmwareVersion) DeviceProfile
}

// KnownProfiles is the registry of all known device profiles.
// Profiles self-register via init() functions in the profiles package.
var KnownProfiles []ProfileEntry

// DetectProfile examines the card name and firmware version to return the appropriate DeviceProfile.
func DetectProfile(card *scarlettctl.Card) (DeviceProfile, error) {
	name := card.Name

	// Get candidate profile entries based on card name
	candidates := getCandidateEntries(name)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("unknown Scarlett device: %s", name)
	}

	// Read firmware info from hardware
	fwInfo, err := ReadFirmwareInfo(card)
	if err != nil {
		return nil, fmt.Errorf("reading firmware version: %w", err)
	}

	// Find profile entry with matching firmware
	for _, entry := range candidates {
		if entry.AppFirmware.Equal(fwInfo.App) && entry.ESPFirmware.Equal(fwInfo.ESP) {
			return entry.CreateProfile(entry.AppFirmware, entry.ESPFirmware), nil
		}
	}

	// No matching profile - build helpful error message
	first := candidates[0]
	return nil, fmt.Errorf(
		"incompatible firmware for %s:\n"+
			"  Detected: App=%s, ESP=%s\n"+
			"  Expected: App=%s, ESP=%s",
		name,
		fwInfo.App, fwInfo.ESP,
		first.AppFirmware, first.ESPFirmware)
}

// getCandidateEntries returns all profile entries that might match the given card name.
func getCandidateEntries(cardName string) []ProfileEntry {
	var entries []ProfileEntry
	for _, entry := range KnownProfiles {
		if strings.Contains(cardName, entry.CardNameContains) {
			entries = append(entries, entry)
		}
	}
	return entries
}
