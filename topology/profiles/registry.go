package profiles

import "github.com/michaelquigley/sessionmixer/topology"

// RegisterProfile adds a profile entry to the known profiles registry.
func RegisterProfile(entry topology.ProfileEntry) {
	topology.KnownProfiles = append(topology.KnownProfiles, entry)
}
