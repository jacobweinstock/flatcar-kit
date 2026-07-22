package main

import (
	"log/slog"
	"sort"
	"testing"
)

func TestBuildRootSubcommands(t *testing.T) {
	root := buildRoot(slog.Default())

	if root.Name != "flatcar-kit" {
		t.Fatalf("root name = %q, want %q", root.Name, "flatcar-kit")
	}

	var names []string
	for _, sub := range root.Subcommands {
		names = append(names, sub.Name)
	}
	sort.Strings(names)

	want := []string{"all", "ignition", "install"}
	if len(names) != len(want) {
		t.Fatalf("subcommands = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("subcommands = %v, want %v", names, want)
		}
	}
}
