package run

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestCommandStdout(t *testing.T) {
	var out bytes.Buffer
	stdin := strings.NewReader("hello world")
	if err := Command(context.Background(), stdin, &out, "cat"); err != nil {
		t.Fatalf("Command() error = %v", err)
	}
	if got := out.String(); got != "hello world" {
		t.Fatalf("stdout = %q, want %q", got, "hello world")
	}
}

func TestCommandNilStdout(t *testing.T) {
	// A nil stdout writer must not panic; the command's output is simply
	// discarded by os/exec.
	if err := Command(context.Background(), nil, nil, "true"); err != nil {
		t.Fatalf("Command() error = %v", err)
	}
}

func TestCommandErrorWrapsStderr(t *testing.T) {
	err := Command(context.Background(), nil, nil, "sh", "-c", "echo boom >&2; exit 3")
	if err == nil {
		t.Fatal("Command() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error = %v, want it to contain captured stderr %q", err, "boom")
	}
	if !strings.Contains(err.Error(), "sh") {
		t.Fatalf("error = %v, want it to contain the command name %q", err, "sh")
	}
}

func TestCommandErrorNoStderr(t *testing.T) {
	err := Command(context.Background(), nil, nil, "sh", "-c", "exit 1")
	if err == nil {
		t.Fatal("Command() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "sh") {
		t.Fatalf("error = %v, want it to contain the command name %q", err, "sh")
	}
}

func TestCommandNotFound(t *testing.T) {
	if err := Command(context.Background(), nil, nil, "flatcar-kit-nonexistent-binary"); err == nil {
		t.Fatal("Command() error = nil, want non-nil for missing binary")
	}
}
