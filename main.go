// Command flatcar-kit is a Tinkerbell action image entrypoint that provides
// Flatcar utilities. It dispatches to one of three modes: ignition, install, or
// all. The mode is selected by a positional subcommand.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jacobweinstock/flatcar-kit/internal/action"
	"github.com/jacobweinstock/flatcar-kit/internal/build"
	"github.com/jacobweinstock/flatcar-kit/internal/config"
	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, done := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
	defer done()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger.Info("flatcar-kit", "version", build.GitRevision())

	root := buildRoot(logger)

	err := root.ParseAndRun(ctx, os.Args[1:], ff.WithEnvVars())
	switch {
	case errors.Is(err, ff.ErrHelp):
		fmt.Fprintln(os.Stderr, ffhelp.Command(root))
		return 0
	case errors.Is(err, ff.ErrNoExec):
		fmt.Fprintln(os.Stderr, ffhelp.Command(root))
		return 2
	case err != nil:
		logger.Error("command failed", "error", err)
		return 1
	}
	return 0
}

// buildRoot constructs the command tree and returns it along with the argument
// slice to parse.
func buildRoot(logger *slog.Logger) *ff.Command {
	var (
		ignCfg     config.Ignition
		instCfg    config.Install
		allIgnCfg  config.Ignition
		allInstCfg config.Install
	)

	ignFlags := ff.NewFlagSet("ignition")
	ignCfg.RegisterFlags(ignFlags)
	ignitionCmd := &ff.Command{
		Name:      "ignition",
		Usage:     "flatcar-kit ignition [FLAGS]",
		ShortHelp: "transpile a Butane config to an Ignition JSON file",
		Flags:     ignFlags,
		Exec: func(ctx context.Context, _ []string) error {
			return action.Ignition(ctx, logger, &ignCfg)
		},
	}

	instFlags := ff.NewFlagSet("install")
	instCfg.RegisterFlags(instFlags, true)
	installCmd := &ff.Command{
		Name:      "install",
		Usage:     "flatcar-kit install [FLAGS]",
		ShortHelp: "write Flatcar to a disk with flatcar-install",
		Flags:     instFlags,
		Exec: func(ctx context.Context, _ []string) error {
			return action.Install(ctx, logger, &instCfg)
		},
	}

	allFlags := ff.NewFlagSet("all")
	allIgnCfg.RegisterFlags(allFlags)
	allInstCfg.RegisterFlags(allFlags, false)
	allCmd := &ff.Command{
		Name:      "all",
		Usage:     "flatcar-kit all [FLAGS]",
		ShortHelp: "run ignition then feed its output to install",
		Flags:     allFlags,
		Exec: func(ctx context.Context, _ []string) error {
			return action.All(ctx, logger, &allIgnCfg, &allInstCfg)
		},
	}

	root := &ff.Command{
		Name:        "flatcar-kit",
		Usage:       "flatcar-kit <ignition|install|all> [FLAGS]",
		ShortHelp:   "Flatcar utilities for Tinkerbell workflows",
		Subcommands: []*ff.Command{ignitionCmd, installCmd, allCmd},
	}

	return root
}
