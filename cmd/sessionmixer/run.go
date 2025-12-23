package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/michaelquigley/dfx"
	"github.com/michaelquigley/scarlettctl"
	"github.com/michaelquigley/sessionmixer"
	"github.com/michaelquigley/sessionmixer/session"
	"github.com/michaelquigley/sessionmixer/topology"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// uiConfig holds persistent UI state (window position, etc.)
type uiConfig struct {
	Window     dfx.WindowConfig    `json:"window"`
	Waterfalls map[string]bool     `json:"waterfalls,omitempty"` // key: "cueMixName/deviceName" or "cueMixName/master"
}

// defaultUIConfig returns sensible defaults for the UI configuration
func defaultUIConfig(numCueMixes int) *uiConfig {
	// Calculate default height based on number of cue mixes
	height := 150 + numCueMixes*420
	if height < 500 {
		height = 500
	}
	if height > 1200 {
		height = 1200
	}

	return &uiConfig{
		Window: dfx.WindowConfig{
			X:      100,
			Y:      100,
			Width:  800,
			Height: height,
		},
	}
}

func init() {
	rootCmd.AddCommand(newRunCommand().cmd)
}

type runCommand struct {
	cmd *cobra.Command
}

func newRunCommand() *runCommand {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run the interactive session mixer",
		Args:  cobra.NoArgs,
	}
	out := &runCommand{cmd: cmd}
	cmd.RunE = out.run
	return out
}

func (cmd *runCommand) run(_ *cobra.Command, _ []string) error {
	// Load session configuration
	cfg, err := session.LoadMainConfig()
	if err != nil {
		return errors.Wrap(err, "error loading session config")
	}

	// Open ALSA card
	card, err := scarlettctl.OpenCard(cfg.Card)
	if err != nil {
		return errors.Wrapf(err, "error opening card '%d'", cfg.Card)
	}
	defer card.Close()

	// Detect device profile from hardware
	profile, err := topology.DetectProfile(card)
	if err != nil {
		return errors.Wrap(err, "error detecting device profile")
	}

	// Build device topology
	builder := topology.NewDeviceBuilder(card, profile)
	device, err := builder.Build()
	if err != nil {
		return errors.Wrap(err, "error building device topology")
	}

	// Create device state for runtime tracking
	state := topology.NewDeviceState(device)

	// Build session from configuration
	sessionBuilder := session.NewSessionBuilder(cfg, device, state, card)
	sess, err := sessionBuilder.Build()
	if err != nil {
		return errors.Wrap(err, "error building session")
	}

	// Initialize faders and routing state from hardware values
	for _, cueMix := range sess.CueMixes {
		if err := cueMix.InitializeFaders(); err != nil {
			return errors.Wrap(err, "error initializing faders")
		}

		// Initialize routing state from hardware (respects existing routing)
		if err := cueMix.InitializeFromHardware(); err != nil {
			return errors.Wrapf(err, "error initializing routing for %s", cueMix.Name())
		}
	}

	// Load UI configuration (window position/size)
	home, err := os.UserHomeDir()
	if err != nil {
		return errors.Wrap(err, "error getting user home directory")
	}
	uiCfgPath := filepath.Join(home, ".config", "sessionmixer", "ui.json")

	uiCfg := defaultUIConfig(len(sess.CueMixes))
	if err := dfx.LoadJSON(uiCfgPath, uiCfg); err != nil {
		fmt.Printf("warning: error loading UI config: %v\n", err)
	}

	// Create mixer UI with saved waterfall states
	mixer := sessionmixer.NewSessionMixer(sess, uiCfg.Waterfalls)

	// Start event monitor for hardware change notifications
	faders := mixer.GetAllDeviceFaders()
	monitor := sessionmixer.NewSessionEventMonitor(card, faders)
	if err := monitor.Start(); err != nil {
		return errors.Wrap(err, "error starting event monitor")
	}
	defer monitor.Stop()

	mixer.SetMonitor(monitor)

	app := dfx.New(mixer, dfx.Config{
		Title:  "SessionMixer",
		Width:  uiCfg.Window.Width,
		Height: uiCfg.Window.Height,
		X:      uiCfg.Window.X,
		Y:      uiCfg.Window.Y,

		OnClose: func(app *dfx.App) {
			// Save window state and waterfall states on close
			uiCfg.Window = dfx.CaptureWindowState(app)
			uiCfg.Waterfalls = mixer.GetWaterfallStates()
			if err := dfx.SaveJSON(uiCfgPath, uiCfg); err != nil {
				fmt.Printf("warning: error saving UI config: %v\n", err)
			}
		},
	})
	return app.Run()
}
