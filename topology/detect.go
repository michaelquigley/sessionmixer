package topology

import (
	"fmt"
	"strings"

	"github.com/michaelquigley/scarlettctl"
)

// DetectProfile examines the card name and returns the appropriate DeviceProfile.
func DetectProfile(card *scarlettctl.Card) (DeviceProfile, error) {
	name := card.Name

	switch {
	case strings.Contains(name, "18i20"):
		return NewScarlett18i20Gen4Profile(), nil
	case strings.Contains(name, "16i16"):
		return NewScarlett16i16Gen4Profile(), nil
	}

	return nil, fmt.Errorf("unknown Scarlett device: %s", name)
}
