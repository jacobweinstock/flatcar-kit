package action

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/jacobweinstock/flatcar-kit/internal/config"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// stubRuns replaces ignitionRun and installRun for the duration of a test.
func stubRuns(t *testing.T, ignPath string, ignErr, instErr error, gotInstall **config.Install) {
	t.Helper()
	origIgn, origInst := ignitionRun, installRun
	t.Cleanup(func() { ignitionRun, installRun = origIgn, origInst })

	ignitionRun = func(_ context.Context, _ *slog.Logger, _ *config.Ignition) (string, error) {
		return ignPath, ignErr
	}
	installRun = func(_ context.Context, _ *slog.Logger, c *config.Install) error {
		if gotInstall != nil {
			*gotInstall = c
		}
		return instErr
	}
}

func TestAllChainsIgnitionOutputToInstall(t *testing.T) {
	var got *config.Install
	stubRuns(t, "/tmp/out/ignition.json", nil, nil, &got)

	ic := &config.Ignition{ButaneConfig: "x"}
	inst := &config.Install{Device: "/dev/sda", IgnitionConfig: "should-be-cleared"}
	if err := All(context.Background(), discardLogger(), ic, inst); err != nil {
		t.Fatalf("All() error = %v", err)
	}
	if got == nil {
		t.Fatal("install step was not invoked")
	}
	if got.IgnitionFile != "/tmp/out/ignition.json" {
		t.Fatalf("install IgnitionFile = %q, want the ignition output path", got.IgnitionFile)
	}
	if got.IgnitionConfig != "" {
		t.Fatalf("install IgnitionConfig = %q, want it cleared", got.IgnitionConfig)
	}
}

func TestAllRejectsValidateOnly(t *testing.T) {
	installed := false
	origInst := installRun
	t.Cleanup(func() { installRun = origInst })
	installRun = func(_ context.Context, _ *slog.Logger, _ *config.Install) error {
		installed = true
		return nil
	}

	ic := &config.Ignition{ButaneConfig: "x", ValidateOnly: true}
	err := All(context.Background(), discardLogger(), ic, &config.Install{Device: "/dev/sda"})
	if err == nil {
		t.Fatal("All() error = nil, want error for VALIDATE_ONLY")
	}
	if installed {
		t.Fatal("install step ran despite VALIDATE_ONLY")
	}
}

func TestAllRejectsStdoutOnly(t *testing.T) {
	ic := &config.Ignition{ButaneConfig: "x", StdoutOnly: true}
	err := All(context.Background(), discardLogger(), ic, &config.Install{Device: "/dev/sda"})
	if err == nil {
		t.Fatal("All() error = nil, want error for STDOUT_ONLY")
	}
}

func TestAllErrorsWhenNoOutputFile(t *testing.T) {
	stubRuns(t, "", nil, nil, nil)

	ic := &config.Ignition{ButaneConfig: "x"}
	err := All(context.Background(), discardLogger(), ic, &config.Install{Device: "/dev/sda"})
	if err == nil {
		t.Fatal("All() error = nil, want error when ignition produced no file")
	}
}

func TestAllPropagatesIgnitionError(t *testing.T) {
	stubRuns(t, "", errors.New("transpile boom"), nil, nil)

	ic := &config.Ignition{ButaneConfig: "x"}
	err := All(context.Background(), discardLogger(), ic, &config.Install{Device: "/dev/sda"})
	if err == nil || err.Error() != "transpile boom" {
		t.Fatalf("All() error = %v, want the ignition error", err)
	}
}

func TestAllPropagatesInstallError(t *testing.T) {
	stubRuns(t, "/tmp/ignition.json", nil, errors.New("install boom"), nil)

	ic := &config.Ignition{ButaneConfig: "x"}
	err := All(context.Background(), discardLogger(), ic, &config.Install{Device: "/dev/sda"})
	if err == nil || err.Error() != "install boom" {
		t.Fatalf("All() error = %v, want the install error", err)
	}
}
