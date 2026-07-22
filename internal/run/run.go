// Package run provides a small helper around os/exec for streaming an external
// command's stdout while capturing its stderr for error reporting.
package run

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Command runs name with args. stdin (may be nil) is connected to the command's
// standard input, and stdout (may be nil) receives the command's standard
// output. Standard error is streamed to os.Stderr so long-running commands show
// progress live, while also being captured so that on a non-zero exit the
// returned error wraps the exit error together with the captured stderr text.
func Command(ctx context.Context, stdin io.Reader, stdout io.Writer, name string, args ...string) error {
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return fmt.Errorf("%s: %w: %s", name, err, msg)
		}
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}
