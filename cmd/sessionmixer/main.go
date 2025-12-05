package main

import (
	"context"
	"log/slog"

	"github.com/charmbracelet/fang"
	"github.com/michaelquigley/df/dl"
	"github.com/spf13/cobra"
)

func init() {
	dl.Init(dl.DefaultOptions().SetLevel(slog.LevelInfo).SetTrimPrefix("github.com/michaelquigley/"))
}

var rootCmd = &cobra.Command{
	Use: "sessionmixer",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if verbose {
			dl.Init(dl.DefaultOptions().SetLevel(slog.LevelDebug).SetTrimPrefix("github.com/michaelquigley/"))
		}
	},
}
var verbose bool

func main() {
	if err := fang.Execute(context.Background(), rootCmd, fang.WithoutManpage(), fang.WithoutCompletions(), fang.WithoutVersion()); err != nil {
		dl.Fatal(err)
	}
}
