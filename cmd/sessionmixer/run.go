package main

import (
	"github.com/michaelquigley/sessionmixer"
	"github.com/michaelquigley/dfx"
	"github.com/michaelquigley/scarlettctl"
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
	cfg, err := sessionmixer.LoadMainConfig()
	if err != nil {
		return err
	}

	card, err := scarlettctl.OpenCard(cfg.Card)
	if err != nil {
		return errors.Wrapf(err, "error opening card '%d'", cfg.Card)
	}
	defer card.Close()

	mapper := sessionmixer.NewControlMapper(card, cfg)
	gangs, err := mapper.LoadGangs()
	if err != nil {
		return errors.Wrap(err, "error loading gangs")
	}

	monitor := sessionmixer.NewEventMonitor(card, gangs)
	if err := monitor.Start(); err != nil {
		return errors.Wrap(err, "error starting event monitor")
	}
	defer monitor.Stop()

	mixer := sessionmixer.NewSessionMixer(card, cfg, gangs)
	app := dfx.New(mixer, dfx.Config{
		Title:  "SessionMixer",
		Width:  530,
		Height: 370,
	})
	return app.Run()
}
