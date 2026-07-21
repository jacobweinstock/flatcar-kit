package config

import "testing"

func TestIgnitionValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Ignition
		wantErr bool
	}{
		{
			name:    "missing butane config",
			cfg:     Ignition{},
			wantErr: true,
		},
		{
			name: "valid",
			cfg:  Ignition{ButaneConfig: "variant: fcos"},
		},
		{
			name:    "tee and stdout-only conflict",
			cfg:     Ignition{ButaneConfig: "x", TeeToStdout: true, StdoutOnly: true},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestInstallValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Install
		wantErr bool
	}{
		{
			name:    "no device or smallest",
			cfg:     Install{},
			wantErr: true,
		},
		{
			name: "device only",
			cfg:  Install{Device: "/dev/sda"},
		},
		{
			name: "smallest only",
			cfg:  Install{InstallToSmallest: true},
		},
		{
			name:    "device and smallest conflict",
			cfg:     Install{Device: "/dev/sda", InstallToSmallest: true},
			wantErr: true,
		},
		{
			name:    "ignition file and config conflict",
			cfg:     Install{Device: "/dev/sda", IgnitionFile: "/a.json", IgnitionConfig: "{}"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
