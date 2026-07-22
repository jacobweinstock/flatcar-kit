// Package install wraps the flatcar-install script: it assembles the command
// line arguments from a typed Config and runs the tool.
package install

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/jacobweinstock/flatcar-kit/internal/config"
	"github.com/jacobweinstock/flatcar-kit/internal/run"
)

// flatcarInstall is the name of the external installer script.
const flatcarInstall = "flatcar-install"

// execCommand runs an external command. It is a package variable so tests can
// substitute a stub in place of the real flatcar-install script.
var execCommand = run.Command

// Args builds the flatcar-install argument slice from c. When an inline Ignition
// config is provided it is written to a temporary file whose path is used with
// the -i flag; the returned cleanup function removes that temp file and must be
// called by the caller (it is a no-op when no temp file was created).
func Args(c *config.Install) (args []string, cleanup func(), err error) {
	cleanup = func() {}

	if c.Device != "" {
		args = append(args, "-d", c.Device)
	}
	if c.InstallToSmallest {
		args = append(args, "-s")
	}

	ignitionPath := c.IgnitionFile
	if c.IgnitionConfig != "" {
		tmp, terr := os.CreateTemp("", "ignition-*.json")
		if terr != nil {
			return nil, cleanup, fmt.Errorf("failed to create temp ignition file: %w", terr)
		}
		if _, werr := tmp.WriteString(c.IgnitionConfig); werr != nil {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
			return nil, cleanup, fmt.Errorf("failed to write temp ignition file: %w", werr)
		}
		_ = tmp.Close()
		ignitionPath = tmp.Name()
		cleanup = func() { _ = os.Remove(tmp.Name()) }
	}
	if ignitionPath != "" {
		args = append(args, "-i", ignitionPath)
	}

	for _, kv := range []struct {
		flag string
		val  string
	}{
		{"-C", c.Channel},
		{"-V", c.Version},
		{"-B", c.Board},
		{"-o", c.OEM},
		{"-b", c.BaseURL},
		{"-k", c.Keyfile},
		{"-f", c.ImageFile},
	} {
		if kv.val != "" {
			args = append(args, kv.flag, kv.val)
		}
	}

	if c.CopyNet {
		args = append(args, "-n")
	}
	if c.CreateUEFI {
		args = append(args, "-u")
	}
	if c.DryRun {
		args = append(args, "-y")
	}
	if c.DownloadOnly {
		args = append(args, "-D")
	}

	args = append(args, strings.Fields(c.ExtraArgs)...)

	return args, cleanup, nil
}

// Run assembles the flatcar-install arguments from c and runs the installer,
// streaming its output to stdout.
func Run(ctx context.Context, logger *slog.Logger, c *config.Install) error {
	if err := c.Validate(); err != nil {
		return err
	}

	args, cleanup, err := Args(c)
	if err != nil {
		return err
	}
	defer cleanup()

	logger.Info("Running flatcar-install...")
	if err := execCommand(ctx, nil, os.Stdout, flatcarInstall, args...); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}
	logger.Info("Install successful.")
	return nil
}
