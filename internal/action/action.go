// Package action wires the three flatcar-kit modes together: ignition,
// install, and all (ignition followed by install).
package action

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jacobweinstock/flatcar-kit/internal/config"
	"github.com/jacobweinstock/flatcar-kit/internal/ignition"
	"github.com/jacobweinstock/flatcar-kit/internal/install"
)

// ignitionRun and installRun are indirections over the ignition and install
// entrypoints so tests can stub the external tool steps.
var (
	ignitionRun = ignition.Run
	installRun  = install.Run
)

// Ignition transpiles a Butane config to an Ignition file.
func Ignition(ctx context.Context, logger *slog.Logger, c *config.Ignition) error {
	_, err := ignitionRun(ctx, logger, c)
	return err
}

// Install writes Flatcar to disk with flatcar-install.
func Install(ctx context.Context, logger *slog.Logger, c *config.Install) error {
	return installRun(ctx, logger, c)
}

// All runs the ignition mode and then feeds its output to the install mode. The
// Ignition file produced by the ignition step becomes the install step's
// Ignition input, overriding any IGNITION_FILE / IGNITION_CONFIG values.
func All(ctx context.Context, logger *slog.Logger, ic *config.Ignition, inst *config.Install) error {
	// In "all" mode the ignition step must produce a file to hand to install.
	// ValidateOnly and StdoutOnly both cause ignition.Run to write nothing,
	// which would otherwise leave install running with no Ignition config.
	if ic.ValidateOnly {
		return errors.New("VALIDATE_ONLY cannot be used in all mode: the install step needs the transpiled Ignition file")
	}
	if ic.StdoutOnly {
		return errors.New("STDOUT_ONLY cannot be used in all mode: the install step needs the transpiled Ignition file")
	}

	outPath, err := ignitionRun(ctx, logger, ic)
	if err != nil {
		return err
	}
	if outPath == "" {
		return errors.New("ignition step produced no output file for the install step")
	}
	inst.IgnitionFile = outPath
	inst.IgnitionConfig = ""
	return installRun(ctx, logger, inst)
}
