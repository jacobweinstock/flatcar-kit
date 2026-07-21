// Package action wires the three flatcar-kit modes together: ignition,
// install, and all (ignition followed by install).
package action

import (
	"context"
	"log/slog"

	"github.com/jacobweinstock/flatcar-kit/internal/config"
	"github.com/jacobweinstock/flatcar-kit/internal/ignition"
	"github.com/jacobweinstock/flatcar-kit/internal/install"
)

// Ignition transpiles a Butane config to an Ignition file.
func Ignition(ctx context.Context, logger *slog.Logger, c *config.Ignition) error {
	_, err := ignition.Run(ctx, logger, c)
	return err
}

// Install writes Flatcar to disk with flatcar-install.
func Install(ctx context.Context, logger *slog.Logger, c *config.Install) error {
	return install.Run(ctx, logger, c)
}

// All runs the ignition mode and then feeds its output to the install mode. The
// Ignition file produced by the ignition step becomes the install step's
// Ignition input, overriding any IGNITION_FILE / IGNITION_CONFIG values.
func All(ctx context.Context, logger *slog.Logger, ic *config.Ignition, inst *config.Install) error {
	outPath, err := ignition.Run(ctx, logger, ic)
	if err != nil {
		return err
	}
	inst.IgnitionFile = outPath
	inst.IgnitionConfig = ""
	return install.Run(ctx, logger, inst)
}
