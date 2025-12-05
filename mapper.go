package sessionmixer

import (
	"fmt"

	"github.com/michaelquigley/scarlettctl"
)

// ControlMapper handles mapping configuration to hardware controls
type ControlMapper struct {
	card   *scarlettctl.Card
	config *Config
}

// NewControlMapper creates a new control mapper
func NewControlMapper(card *scarlettctl.Card, config *Config) *ControlMapper {
	return &ControlMapper{
		card:   card,
		config: config,
	}
}

// LoadGangs creates GangedFader instances from the config
func (cm *ControlMapper) LoadGangs() ([]*GangedFader, error) {
	var gangs []*GangedFader

	for i, gangControl := range cm.config.GangControls {
		// Find all hardware controls for this gang
		var gangChannels []*MixerChannel

		for j, ctrlName := range gangControl.Controls {
			control, err := cm.card.FindControl(ctrlName)
			if err != nil {
				return nil, fmt.Errorf("gang %d (%s), control %d (%s): not found on hardware: %w", i, gangControl.Name, j, ctrlName, err)
			}

			// Validate control type
			if control.Type != scarlettctl.ControlTypeInteger && control.Type != scarlettctl.ControlTypeInteger64 {
				return nil, fmt.Errorf("gang %d (%s), control %d (%s): type %d not supported", i, gangControl.Name, j, ctrlName, control.Type)
			}

			// Create a display name for the channel within the gang
			displayName := fmt.Sprintf("%s [%s]", gangControl.Name, ctrlName)

			// Create mixer channel
			ch, err := NewMixerChannel(control, displayName, gangControl.Unit)
			if err != nil {
				return nil, fmt.Errorf("gang %d (%s), control %d (%s): failed to create channel: %w", i, gangControl.Name, j, ctrlName, err)
			}

			gangChannels = append(gangChannels, ch)
		}

		// Find level controls for this gang (optional)
		var levelControls []*scarlettctl.Control
		for j, levelName := range gangControl.Levels {
			levelCtl, err := cm.card.FindControl(levelName)
			if err != nil {
				return nil, fmt.Errorf("gang %d (%s), level %d (%s): not found on hardware: %w",
					i, gangControl.Name, j, levelName, err)
			}
			levelControls = append(levelControls, levelCtl)
		}

		// Create ganged fader (mirror mode only for now)
		gang, err := NewGangedFader(gangControl.Name, gangControl.Unit, GangModeMirror, gangChannels, levelControls, gangControl.TaperDb)
		if err != nil {
			return nil, fmt.Errorf("gang %d (%s): failed to create ganged fader: %w", i, gangControl.Name, err)
		}

		gangs = append(gangs, gang)
	}

	return gangs, nil
}
