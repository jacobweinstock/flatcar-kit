// Package config defines the typed configuration for each flatcar-kit mode
// and registers those fields as ff flags. Environment variable fallback is
// handled by ff.WithEnvVars (no prefix), so a flag named "butane-config" is
// populated from the BUTANE_CONFIG environment variable.
package config

import (
	"errors"

	"github.com/peterbourgon/ff/v4"
)

// Ignition holds configuration for the "ignition" mode (Butane -> Ignition).
type Ignition struct {
	ButaneConfig string // BUTANE_CONFIG
	TargetPath   string // TARGET_PATH
	TargetFile   string // TARGET_FILE
	ValidateOnly bool   // VALIDATE_ONLY
	ExtraArgs    string // EXTRA_ARGS
	TeeToStdout  bool   // TEE_TO_STDOUT
	StdoutOnly   bool   // STDOUT_ONLY
}

// RegisterFlags registers the ignition flags on fs. Flag long names are the
// kebab-case of their environment variable name.
func (c *Ignition) RegisterFlags(fs *ff.FlagSet) {
	fs.StringVar(&c.ButaneConfig, 0, "butane-config", "", "Butane config content to transpile (env BUTANE_CONFIG)")
	fs.StringVar(&c.TargetPath, 0, "target-path", "/tmp/", "output directory for the Ignition file (env TARGET_PATH)")
	fs.StringVar(&c.TargetFile, 0, "target-file", "ignition.json", "output file name for the Ignition file (env TARGET_FILE)")
	fs.BoolVar(&c.ValidateOnly, 0, "validate-only", "only validate the Butane config, do not transpile (env VALIDATE_ONLY)")
	fs.StringVar(&c.ExtraArgs, 0, "extra-args", "", "additional arguments passed to butane (env EXTRA_ARGS)")
	fs.BoolVar(&c.TeeToStdout, 0, "tee-to-stdout", "write the Ignition output to the target file and stdout (env TEE_TO_STDOUT)")
	fs.BoolVar(&c.StdoutOnly, 0, "stdout-only", "write the Ignition output to stdout only (env STDOUT_ONLY)")
}

// Validate checks the ignition configuration for required and conflicting values.
func (c *Ignition) Validate() error {
	if c.ButaneConfig == "" {
		return errors.New("BUTANE_CONFIG (butane-config) is required")
	}
	if c.TeeToStdout && c.StdoutOnly {
		return errors.New("TEE_TO_STDOUT and STDOUT_ONLY are mutually exclusive")
	}
	return nil
}

// Install holds configuration for the "install" mode. Fields map to
// flatcar-install command-line flags.
type Install struct {
	Device            string // DEVICE            -> -d
	InstallToSmallest bool   // INSTALL_TO_SMALLEST -> -s
	IgnitionFile      string // IGNITION_FILE     -> -i
	IgnitionConfig    string // IGNITION_CONFIG   -> inline, written to temp file, -i
	Channel           string // CHANNEL           -> -C
	Version           string // VERSION           -> -V
	Board             string // BOARD             -> -B
	OEM               string // OEM               -> -o
	BaseURL           string // BASE_URL          -> -b
	Keyfile           string // KEYFILE           -> -k
	ImageFile         string // IMAGE_FILE        -> -f
	CopyNet           bool   // COPY_NET          -> -n
	CreateUEFI        bool   // CREATE_UEFI       -> -u
	DryRun            bool   // DRY_RUN           -> -y
	DownloadOnly      bool   // DOWNLOAD_ONLY     -> -D
	ExtraArgs         string // EXTRA_ARGS        -> passthrough
}

// RegisterFlags registers the install flags on fs. When withExtraArgs is false
// the shared "extra-args" (EXTRA_ARGS) flag is not registered; this is used by
// the "all" command, where the ignition config owns that flag to avoid a
// duplicate flag-name collision on the shared flag set.
func (c *Install) RegisterFlags(fs *ff.FlagSet, withExtraArgs bool) {
	fs.StringVar(&c.Device, 0, "device", "", "target block device, e.g. /dev/sda (env DEVICE, -d)")
	fs.BoolVar(&c.InstallToSmallest, 0, "install-to-smallest", "install to the smallest available disk (env INSTALL_TO_SMALLEST, -s)")
	fs.StringVar(&c.IgnitionFile, 0, "ignition-file", "", "path to an Ignition config file (env IGNITION_FILE, -i)")
	fs.StringVar(&c.IgnitionConfig, 0, "ignition-config", "", "inline Ignition config, written to a temp file (env IGNITION_CONFIG)")
	fs.StringVar(&c.Channel, 0, "channel", "", "Flatcar release channel (env CHANNEL, -C)")
	fs.StringVar(&c.Version, 0, "version", "", "Flatcar version (env VERSION, -V)")
	fs.StringVar(&c.Board, 0, "board", "", "target board (env BOARD, -B)")
	fs.StringVar(&c.OEM, 0, "oem", "", "OEM id (env OEM, -o)")
	fs.StringVar(&c.BaseURL, 0, "base-url", "", "base URL for image download (env BASE_URL, -b)")
	fs.StringVar(&c.Keyfile, 0, "keyfile", "", "GPG key file for signature verification (env KEYFILE, -k)")
	fs.StringVar(&c.ImageFile, 0, "image-file", "", "local image file to install (env IMAGE_FILE, -f)")
	fs.BoolVar(&c.CopyNet, 0, "copy-net", "copy network units to the installed system (env COPY_NET, -n)")
	fs.BoolVar(&c.CreateUEFI, 0, "create-uefi", "create a UEFI boot entry (env CREATE_UEFI, -u)")
	fs.BoolVar(&c.DryRun, 0, "dry-run", "print the flatcar-install invocation without writing to disk (env DRY_RUN, -y)")
	fs.BoolVar(&c.DownloadOnly, 0, "download-only", "download the image only, do not install (env DOWNLOAD_ONLY, -D)")
	if withExtraArgs {
		fs.StringVar(&c.ExtraArgs, 0, "extra-args", "", "additional arguments passed to flatcar-install (env EXTRA_ARGS)")
	}
}

// Validate checks the install configuration for required and conflicting values.
func (c *Install) Validate() error {
	if c.Device == "" && !c.InstallToSmallest {
		return errors.New("one of DEVICE (device) or INSTALL_TO_SMALLEST (install-to-smallest) is required")
	}
	if c.Device != "" && c.InstallToSmallest {
		return errors.New("DEVICE and INSTALL_TO_SMALLEST are mutually exclusive")
	}
	if c.IgnitionFile != "" && c.IgnitionConfig != "" {
		return errors.New("IGNITION_FILE and IGNITION_CONFIG are mutually exclusive")
	}
	return nil
}
