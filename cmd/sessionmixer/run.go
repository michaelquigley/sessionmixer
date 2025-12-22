package main

import (
	"github.com/michaelquigley/dfx"
	"github.com/michaelquigley/scarlettctl"
	"github.com/michaelquigley/sessionmixer"
	"github.com/michaelquigley/sessionmixer/session"
	"github.com/michaelquigley/sessionmixer/topology"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

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

	// Initialize faders from hardware values
	for _, cueMix := range sess.CueMixes {
		if err := cueMix.InitializeFaders(); err != nil {
			return errors.Wrap(err, "error initializing faders")
		}

		// Setup routing for each cue mix
		if err := cueMix.SetupRouting(); err != nil {
			return errors.Wrapf(err, "error setting up routing for %s", cueMix.Name())
		}
	}

	// Create mixer UI
	mixer := sessionmixer.NewSessionMixer(sess)

	// Start event monitor for hardware change notifications
	faders := mixer.GetAllDeviceFaders()
	monitor := sessionmixer.NewSessionEventMonitor(card, faders)
	if err := monitor.Start(); err != nil {
		return errors.Wrap(err, "error starting event monitor")
	}
	defer monitor.Stop()

	mixer.SetMonitor(monitor)

	// Calculate window size based on number of cue mixes
	// Base height per cue mix + some padding
	height := 150 + len(sess.CueMixes)*420
	if height < 500 {
		height = 500
	}
	if height > 1200 {
		height = 1200
	}

	app := dfx.New(mixer, dfx.Config{
		Title:  "SessionMixer",
		Width:  800,
		Height: height,
	})
	return app.Run()
}
