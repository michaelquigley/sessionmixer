package main

import (
	"fmt"
	"strings"

	"github.com/michaelquigley/scarlettctl"
	"github.com/michaelquigley/sessionmixer"
	"github.com/michaelquigley/sessionmixer/topology"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newTopologyCommand().cmd)
}

type topologyCommand struct {
	cmd         *cobra.Command
	showRouting bool
	showMixes   bool
	showMeters  bool
}

func newTopologyCommand() *topologyCommand {
	cmd := &cobra.Command{
		Use:   "topology",
		Short: "Display discovered device topology",
		Args:  cobra.NoArgs,
	}
	out := &topologyCommand{cmd: cmd}
	cmd.Flags().BoolVarP(&out.showRouting, "routing", "r", false, "Show current routing")
	cmd.Flags().BoolVarP(&out.showMixes, "mixes", "m", false, "Show mix details")
	cmd.Flags().BoolVarP(&out.showMeters, "meters", "l", false, "Show level meters")
	cmd.RunE = out.run
	return out
}

func (cmd *topologyCommand) run(_ *cobra.Command, _ []string) error {
	cfg, err := sessionmixer.LoadMainConfig()
	if err != nil {
		return err
	}

	card, err := scarlettctl.OpenCard(cfg.Card)
	if err != nil {
		return errors.Wrapf(err, "error opening card '%d'", cfg.Card)
	}
	defer card.Close()

	// Read firmware info from hardware (for display)
	fwInfo, err := topology.ReadFirmwareInfo(card)
	if err != nil {
		return errors.Wrap(err, "error reading firmware version")
	}

	// Detect device profile
	profile, err := topology.DetectProfile(card)
	if err != nil {
		return errors.Wrap(err, "device detection failed")
	}
	builder := topology.NewDeviceBuilder(card, profile)
	device, err := builder.Build()
	if err != nil {
		return errors.Wrap(err, "error building topology")
	}

	// Create state for level meters
	state := topology.NewDeviceState(device)

	// Display summary
	fmt.Printf("Device: %s\n", device.Name)
	fmt.Printf("Profile: %s\n", profile.Name())
	fmt.Printf("Card: %d\n", cfg.Card)
	fmt.Printf("Firmware: App=%s, ESP=%s\n", fwInfo.App, fwInfo.ESP)
	fmt.Println(strings.Repeat("=", 60))

	// Display ports
	cmd.printPorts("Analogue Inputs", device.AnalogueInputs)
	cmd.printPorts("Analogue Outputs", device.AnalogueOutputs)
	cmd.printPorts("S/PDIF Inputs", device.SPDIFInputs)
	cmd.printPorts("S/PDIF Outputs", device.SPDIFOutputs)
	cmd.printPorts("ADAT Inputs", device.ADATInputs)
	cmd.printPorts("ADAT Outputs", device.ADATOutputs)
	cmd.printPorts("PCM Playback", device.PCMPlayback)
	cmd.printPorts("PCM Capture", device.PCMCapture)

	// Display mixes
	if cmd.showMixes {
		cmd.printMixes(device)
	} else {
		fmt.Printf("\nMixes: %d (A-%c)\n", len(device.Mixes), 'A'+len(device.Mixes)-1)
		fmt.Println("  (use --mixes to show details)")
	}

	// Display routing
	if cmd.showRouting {
		cmd.printRouting(device)
	} else {
		totalEndpoints := len(device.AnalogueOutputEndpoints) +
			len(device.SPDIFOutputEndpoints) +
			len(device.ADATOutputEndpoints) +
			len(device.PCMCaptureEndpoints) +
			len(device.MixerInputEndpoints)
		fmt.Printf("\nRouting Endpoints: %d\n", totalEndpoints)
		fmt.Println("  (use --routing to show details)")
	}

	// Display meters
	if cmd.showMeters {
		if err := state.ReadAllLevels(); err != nil {
			fmt.Printf("\nLevel Meters: error reading: %v\n", err)
		} else {
			cmd.printMeters(device, state)
		}
	} else {
		fmt.Printf("\nLevel Meters: %d\n", len(device.LevelMeterControls))
		fmt.Println("  (use --meters to show values)")
	}

	return nil
}

func (cmd *topologyCommand) printPorts(title string, ports []*topology.Port) {
	if len(ports) == 0 {
		return
	}

	fmt.Printf("\n%s (%d):\n", title, len(ports))
	for _, p := range ports {
		caps := []string{}
		if p.HasPhantom {
			caps = append(caps, "phantom")
		}
		if p.HasAir {
			caps = append(caps, "air")
		}
		if p.HasPad {
			caps = append(caps, "pad")
		}
		if p.HasGain {
			caps = append(caps, "gain")
		}

		capStr := ""
		if len(caps) > 0 {
			capStr = fmt.Sprintf(" [%s]", strings.Join(caps, ", "))
		}

		meterStr := ""
		if p.LevelMeterIndex >= 0 {
			meterStr = fmt.Sprintf(" (meter:%d)", p.LevelMeterIndex)
		}

		fmt.Printf("  %-20s %s%s%s\n", p.ID, p.Name, capStr, meterStr)
	}
}

func (cmd *topologyCommand) printMixes(device *topology.Device) {
	fmt.Printf("\nMixes (%d):\n", len(device.Mixes))
	fmt.Println(strings.Repeat("-", 60))

	for _, mix := range device.Mixes {
		inputsWithVolCtl := 0
		for _, input := range mix.Inputs {
			if input.VolumeControl != nil {
				inputsWithVolCtl++
			}
		}

		meterStr := ""
		if mix.OutputLevelMeterIndex >= 0 {
			meterStr = fmt.Sprintf(" (meter:%d)", mix.OutputLevelMeterIndex)
		}

		fmt.Printf("  %s: %d inputs (%d with volume control)%s\n",
			mix.Name, len(mix.Inputs), inputsWithVolCtl, meterStr)
	}

	// Show input mapping for first mix as example
	if len(device.Mixes) > 0 {
		mix := device.Mixes[0]
		fmt.Printf("\n%s Input Mapping:\n", mix.Name)
		for i, input := range mix.Inputs {
			if i > 0 && i%8 == 0 {
				fmt.Println()
			}
			sourceID := "?"
			if input.Source != nil {
				sourceID = input.Source.ShortName
			}
			hasCtl := " "
			if input.VolumeControl != nil {
				hasCtl = "*"
			}
			fmt.Printf("  %2d:%s%-6s", input.InputNumber, hasCtl, sourceID)
		}
		fmt.Println("\n  (* = has volume control)")
	}
}

func (cmd *topologyCommand) printRouting(device *topology.Device) {
	fmt.Println("\nRouting:")
	fmt.Println(strings.Repeat("-", 60))

	printEndpoints := func(title string, endpoints []*topology.RoutingEndpoint) {
		if len(endpoints) == 0 {
			return
		}
		fmt.Printf("\n%s:\n", title)
		for _, ep := range endpoints {
			source := ep.GetCurrentSource()
			fmt.Printf("  %-25s <- %s\n", ep.Port.Name, source)
		}
	}

	printEndpoints("Analogue Outputs", device.AnalogueOutputEndpoints)
	printEndpoints("S/PDIF Outputs", device.SPDIFOutputEndpoints)
	printEndpoints("ADAT Outputs", device.ADATOutputEndpoints)
	printEndpoints("PCM Capture", device.PCMCaptureEndpoints)
	printEndpoints("Mixer Inputs", device.MixerInputEndpoints)
}

func (cmd *topologyCommand) printMeters(device *topology.Device, state *topology.DeviceState) {
	fmt.Println("\nLevel Meters:")
	fmt.Println(strings.Repeat("-", 60))

	// Print meters by category
	printPortMeters := func(title string, ports []*topology.Port) {
		var hasMeters bool
		for _, p := range ports {
			if p.LevelMeterIndex >= 0 {
				hasMeters = true
				break
			}
		}
		if !hasMeters {
			return
		}

		fmt.Printf("\n%s:\n", title)
		for _, p := range ports {
			if p.LevelMeterIndex >= 0 {
				level := state.GetLevelMeter(p.LevelMeterIndex)
				bar := cmd.levelBar(level)
				fmt.Printf("  [%2d] %-20s %6d %s\n", p.LevelMeterIndex, p.ShortName, level, bar)
			}
		}
	}

	printPortMeters("Analogue Inputs", device.AnalogueInputs)
	printPortMeters("S/PDIF Inputs", device.SPDIFInputs)
	printPortMeters("ADAT Inputs", device.ADATInputs)
	printPortMeters("PCM Playback", device.PCMPlayback)

	// Mix output meters
	fmt.Printf("\nMix Outputs:\n")
	for _, mix := range device.Mixes {
		if mix.OutputLevelMeterIndex >= 0 {
			level := state.GetMixOutputLevel(mix.ID)
			bar := cmd.levelBar(level)
			fmt.Printf("  [%2d] %-20s %6d %s\n", mix.OutputLevelMeterIndex, mix.Name, level, bar)
		}
	}

	printPortMeters("Analogue Outputs", device.AnalogueOutputs)
}

func (cmd *topologyCommand) levelBar(level int64) string {
	// Normalize to 0-20 character bar
	// Assuming level is 0-65535 (typical ALSA range)
	const maxLevel = 65535
	const barWidth = 20

	if level <= 0 {
		return strings.Repeat("-", barWidth)
	}

	filled := int(level * barWidth / maxLevel)
	if filled > barWidth {
		filled = barWidth
	}

	return strings.Repeat("#", filled) + strings.Repeat("-", barWidth-filled)
}
