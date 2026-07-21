package install

import (
	"os"
	"reflect"
	"testing"

	"github.com/jacobweinstock/flatcar-kit/internal/config"
)

func TestArgs(t *testing.T) {
	tests := []struct {
		name string
		cfg  config.Install
		want []string
	}{
		{
			name: "device only",
			cfg:  config.Install{Device: "/dev/sda"},
			want: []string{"-d", "/dev/sda"},
		},
		{
			name: "smallest",
			cfg:  config.Install{InstallToSmallest: true},
			want: []string{"-s"},
		},
		{
			name: "ignition file",
			cfg:  config.Install{Device: "/dev/sda", IgnitionFile: "/tmp/ign.json"},
			want: []string{"-d", "/dev/sda", "-i", "/tmp/ign.json"},
		},
		{
			name: "all string flags",
			cfg: config.Install{
				Device:  "/dev/sda",
				Channel: "stable",
				Version: "3000.0.0",
				Board:   "amd64-usr",
				OEM:     "packet",
				BaseURL: "https://example.com",
				Keyfile: "/key.asc",
			},
			want: []string{
				"-d", "/dev/sda",
				"-C", "stable",
				"-V", "3000.0.0",
				"-B", "amd64-usr",
				"-o", "packet",
				"-b", "https://example.com",
				"-k", "/key.asc",
			},
		},
		{
			name: "bool flags and extra args",
			cfg: config.Install{
				Device:       "/dev/sda",
				CopyNet:      true,
				CreateUEFI:   true,
				DryRun:       true,
				DownloadOnly: true,
				ExtraArgs:    "-v --foo",
			},
			want: []string{"-d", "/dev/sda", "-n", "-u", "-y", "-D", "-v", "--foo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, cleanup, err := Args(&tt.cfg)
			if err != nil {
				t.Fatalf("Args() error = %v", err)
			}
			defer cleanup()
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("Args() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestArgsInlineIgnition(t *testing.T) {
	cfg := config.Install{Device: "/dev/sda", IgnitionConfig: `{"ignition":{}}`}

	got, cleanup, err := Args(&cfg)
	if err != nil {
		t.Fatalf("Args() error = %v", err)
	}
	defer cleanup()

	if len(got) != 4 || got[0] != "-d" || got[2] != "-i" {
		t.Fatalf("unexpected args: %v", got)
	}

	tmpPath := got[3]
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatalf("temp ignition file not readable: %v", err)
	}
	if string(data) != cfg.IgnitionConfig {
		t.Fatalf("temp file contents = %q, want %q", string(data), cfg.IgnitionConfig)
	}

	cleanup()
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Fatalf("expected temp file %q to be removed after cleanup", tmpPath)
	}
}
