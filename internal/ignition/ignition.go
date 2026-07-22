// Package ignition ports the Butane transpilation logic from the original
// ignition-gen entrypoint.sh: validate-only, stdout-only, tee, and file output.
package ignition

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/jacobweinstock/flatcar-kit/internal/config"
	"github.com/jacobweinstock/flatcar-kit/internal/run"
)

// butane is the name of the external transpiler binary.
const butane = "butane"

// Run transpiles the Butane config in c to an Ignition JSON document. It returns
// the path of the written Ignition file, or an empty string when the output was
// written to stdout only or when only validation was requested.
func Run(ctx context.Context, logger *slog.Logger, c *config.Ignition) (string, error) {
	if err := c.Validate(); err != nil {
		return "", err
	}

	extra := strings.Fields(c.ExtraArgs)
	stdin := strings.NewReader(c.ButaneConfig)

	if c.ValidateOnly {
		logger.Info("validating Butane config")
		args := append([]string{"--strict"}, extra...)
		if err := run.Command(ctx, stdin, io.Discard, butane, args...); err != nil {
			return "", fmt.Errorf("validation failed: %w", err)
		}
		logger.Info("config valid")
		return "", nil
	}

	logger.Info("transpiling Butane config to Ignition JSON")

	if c.StdoutOnly {
		if err := run.Command(ctx, stdin, os.Stdout, butane, extra...); err != nil {
			return "", fmt.Errorf("transpilation failed: %w", err)
		}
		logger.Info("transpilation successful")
		return "", nil
	}

	if err := os.MkdirAll(c.TargetPath, 0o755); err != nil {
		return "", fmt.Errorf("failed to create destination path %q: %w", c.TargetPath, err)
	}
	outPath := filepath.Join(c.TargetPath, c.TargetFile)

	f, err := os.Create(outPath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file %q: %w", outPath, err)
	}
	defer func() { _ = f.Close() }()

	var stdout io.Writer = f
	if c.TeeToStdout {
		stdout = io.MultiWriter(f, os.Stdout)
	}

	if err := run.Command(ctx, stdin, stdout, butane, extra...); err != nil {
		return "", fmt.Errorf("transpilation failed: %w", err)
	}

	logger.Info("transpilation successful")
	return outPath, nil
}
