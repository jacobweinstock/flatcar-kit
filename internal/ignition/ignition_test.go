package ignition

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jacobweinstock/flatcar-kit/internal/config"
)

// stubExec replaces the package execCommand for the duration of a test. It
// records the command name and args it was called with, writes want to the
// provided stdout writer, and returns err.
func stubExec(t *testing.T, want string, retErr error, gotName *string, gotArgs *[]string) {
	t.Helper()
	orig := execCommand
	t.Cleanup(func() { execCommand = orig })
	execCommand = func(_ context.Context, _ io.Reader, stdout io.Writer, name string, args ...string) error {
		if gotName != nil {
			*gotName = name
		}
		if gotArgs != nil {
			*gotArgs = args
		}
		if retErr != nil {
			return retErr
		}
		if stdout != nil && want != "" {
			_, _ = io.WriteString(stdout, want)
		}
		return nil
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestRunValidateOnly(t *testing.T) {
	var name string
	var args []string
	stubExec(t, "", nil, &name, &args)

	cfg := &config.Ignition{ButaneConfig: "variant: flatcar", ValidateOnly: true, ExtraArgs: "--foo"}
	got, err := Run(context.Background(), discardLogger(), cfg)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got != "" {
		t.Fatalf("Run() path = %q, want empty", got)
	}
	if name != butane {
		t.Fatalf("command = %q, want %q", name, butane)
	}
	if len(args) == 0 || args[0] != "--strict" {
		t.Fatalf("args = %v, want to start with --strict", args)
	}
	if args[len(args)-1] != "--foo" {
		t.Fatalf("args = %v, want to include extra arg --foo", args)
	}
}

func TestRunValidateOnlyError(t *testing.T) {
	stubExec(t, "", errors.New("bad config"), nil, nil)

	cfg := &config.Ignition{ButaneConfig: "x", ValidateOnly: true}
	_, err := Run(context.Background(), discardLogger(), cfg)
	if err == nil || !strings.Contains(err.Error(), "validation failed") {
		t.Fatalf("Run() error = %v, want validation failed", err)
	}
}

func TestRunStdoutOnly(t *testing.T) {
	stubExec(t, `{"ignition":{}}`, nil, nil, nil)

	cfg := &config.Ignition{ButaneConfig: "x", StdoutOnly: true}
	got, err := Run(context.Background(), discardLogger(), cfg)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got != "" {
		t.Fatalf("Run() path = %q, want empty for stdout-only", got)
	}
}

func TestRunFileOutput(t *testing.T) {
	stubExec(t, `{"ignition":{}}`, nil, nil, nil)

	dir := filepath.Join(t.TempDir(), "nested", "out")
	cfg := &config.Ignition{ButaneConfig: "x", TargetPath: dir, TargetFile: "ignition.json"}
	got, err := Run(context.Background(), discardLogger(), cfg)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	want := filepath.Join(dir, "ignition.json")
	if got != want {
		t.Fatalf("Run() path = %q, want %q", got, want)
	}
	data, err := os.ReadFile(want)
	if err != nil {
		t.Fatalf("output file not readable: %v", err)
	}
	if string(data) != `{"ignition":{}}` {
		t.Fatalf("file contents = %q, want %q", string(data), `{"ignition":{}}`)
	}
}

func TestRunTeeToStdout(t *testing.T) {
	stubExec(t, `{"ignition":{}}`, nil, nil, nil)

	dir := t.TempDir()
	cfg := &config.Ignition{ButaneConfig: "x", TargetPath: dir, TargetFile: "ignition.json", TeeToStdout: true}
	got, err := Run(context.Background(), discardLogger(), cfg)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	data, err := os.ReadFile(got)
	if err != nil {
		t.Fatalf("output file not readable: %v", err)
	}
	if string(data) != `{"ignition":{}}` {
		t.Fatalf("file contents = %q, want %q", string(data), `{"ignition":{}}`)
	}
}

func TestRunTranspileError(t *testing.T) {
	stubExec(t, "", errors.New("boom"), nil, nil)

	cfg := &config.Ignition{ButaneConfig: "x", TargetPath: t.TempDir(), TargetFile: "ignition.json"}
	_, err := Run(context.Background(), discardLogger(), cfg)
	if err == nil || !strings.Contains(err.Error(), "transpilation failed") {
		t.Fatalf("Run() error = %v, want transpilation failed", err)
	}
}

func TestRunValidationRejectsMissingConfig(t *testing.T) {
	_, err := Run(context.Background(), discardLogger(), &config.Ignition{})
	if err == nil {
		t.Fatal("Run() error = nil, want required-config error")
	}
}
