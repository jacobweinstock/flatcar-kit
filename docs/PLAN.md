# Implementation Plan: `flatcar-kit` — unified Flatcar Tinkerbell action image

A single container image that provides Flatcar-related utilities for Tinkerbell
workflows. A Go orchestrator binary is the entrypoint; it dispatches to one of
three modes and execs the external tools (`butane` binary, `flatcar-install`
bash script), streaming and logging their output.

This plan is self-contained so it can be implemented in a fresh VS Code / chat
session. It originates from the existing `ignition-gen` repo
(`../ignition-gen/entrypoint.sh` + `Dockerfile`), which is being generalized.

---

## Goals / modes

Three modes, selected by a positional subcommand OR the `ACTION` env var
(subcommand wins; if absent, fall back to `ACTION`):

1. `ignition` — transpile a Butane config to an Ignition JSON (today's behavior).
2. `install`  — wrap `flatcar-install` to write Flatcar to a disk.
3. `all`      — run `ignition`, then feed its output to `install`.

Built to run as a **Tinkerbell Action**:
- Config comes only from env vars (Tinkerbell `environment` map).
- Structured JSON logs to stdout.
- Exit 0 on success, non-zero on failure.

---

## Key decisions

- **Language: Go** (not Bash). Rationale: 3 modes + inline/file input + non-trivial
  arg assembly + testability; matches the Tinkerbell (Go) ecosystem; single static
  binary; `log/slog` JSON logging; `os/exec` handles stdout/stderr/exit codes cleanly
  (this retires the Bash tee/`pipefail`/stderr-capture problems of the old script).
- **External tools stay external** — Go only orchestrates `butane` and `flatcar-install`.
- **Module**: `github.com/jacobweinstock/flatcar-kit`; **binary**: `flatcar-kit`.
- **CLI/env parsing**: `github.com/peterbourgon/ff/v4`.
  - Use `ff.Command` subcommands for `ignition | install | all`.
  - Each command has its own `ff.FlagSet`.
  - Env fallback via `ff.WithEnvVars()` with **NO prefix**, so existing env names are
    preserved (`BUTANE_CONFIG`, `TARGET_PATH`, `DEVICE`, ...). Flag names are the
    kebab-case of the env name (`butane-config` <-> `BUTANE_CONFIG`).
  - Mode selection: positional subcommand; if none given, inject `ACTION` env as the command.
  - `ffhelp` for usage; unknown/no command -> usage + exit 2.
- **Ignition input to `install`**: accept a file path (`IGNITION_FILE`) OR inline
  content (`IGNITION_CONFIG`, written to a temp file).
- **Pin `flatcar-install`** to a specific `flatcar/init` commit SHA (not `flatcar-master`).

---

## Target repo layout

```
go.mod / go.sum                 module github.com/jacobweinstock/flatcar-kit;
                                dep: github.com/peterbourgon/ff/v4
main.go                         build ff.Command tree (root + ignition|install|all);
                                if no positional cmd, inject ACTION env as the command;
                                Parse, Run, map errors to exit codes
internal/config/config.go       Config structs per mode; registered as ff flags
                                (kebab-case) with ff.WithEnvVars() (no prefix); validation
internal/log/log.go             slog JSON handler (timestamp, level, message) to stdout
internal/run/run.go             exec helper: run cmd, stream stdout, capture stderr, return err
internal/ignition/ignition.go   butane validate/transpile (validate-only, tee, stdout-only, file)
internal/install/install.go     assemble flatcar-install args from Config; inline/file ignition
internal/action/action.go       ff.Command handlers: ignition | install | all
*_test.go                       table-driven tests for config parsing + arg assembly
Dockerfile                      multi-stage: build Go binary; final = butane base +
                                flatcar-install + toolset + copied binary
```

---

## Env var contract (parsed into typed Config)

- **Common**: `ACTION=ignition|install|all` (or positional subcommand).
- **ignition**:
  - `BUTANE_CONFIG` (required)
  - `TARGET_PATH` (default `/tmp/`)
  - `TARGET_FILE` (default `ignition.json`)
  - `VALIDATE_ONLY` (bool)
  - `EXTRA_ARGS`
  - `TEE_TO_STDOUT` (bool)
  - `STDOUT_ONLY` (bool)
- **install** (maps to `flatcar-install` flags):
  - `DEVICE` (-d) **or** `INSTALL_TO_SMALLEST` (-s)
  - `IGNITION_FILE` (-i) **or** `IGNITION_CONFIG` (inline -> temp file -> -i)
  - `CHANNEL` (-C), `VERSION` (-V), `BOARD` (-B), `OEM` (-o), `BASE_URL` (-b),
    `KEYFILE` (-k), `IMAGE_FILE` (-f)
  - `COPY_NET` (-n bool), `CREATE_UEFI` (-u bool), `DRY_RUN` (-y bool),
    `DOWNLOAD_ONLY` (-D bool)
  - `EXTRA_ARGS` (passthrough)
- **all**: `ignition` runs first; its output path becomes `install`'s ignition input
  automatically (default `/tmp/ignition.json`).

---

## `flatcar-install` facts (researched)

- Requires **root** + block-device access; run **privileged**, mount `/dev`.
- Required toolset: `blockdev btrfstune cp cut dd gawk gpg grep head ls lsblk lvm
  mkdir mkfifo mktemp mount rm sed sort tee udevadm wget wipefs` + `bzip2`/`lbzip2`;
  `efibootmgr` only for `-u` (UEFI).
- Key flags: `-d` dev, `-s` smallest, `-D` download-only, `-i` ignition, `-c` cloud-init,
  `-C` channel, `-V` version, `-B` board, `-o` oem, `-b` baseurl, `-k` keyfile,
  `-f` image, `-n` copy-net, `-u` uefi, `-y` dry-run, `-v` verbose.
- Source (pin a commit SHA of this):
  `https://raw.githubusercontent.com/flatcar/init/<REF>/bin/flatcar-install`

---

## Implementation steps

1. Scaffold Go module (`go.mod`, module `github.com/jacobweinstock/flatcar-kit`);
   add `github.com/peterbourgon/ff/v4`; create `main.go` + `internal/` packages.
2. `internal/log`: slog JSON handler matching the current shape
   `{timestamp, level, message}`.
3. `internal/config`: typed structs; register fields as ff flags (kebab-case) per
   command; `ff.WithEnvVars()` no prefix -> existing env names; validation
   (required fields; mutually exclusive `DEVICE` vs `INSTALL_TO_SMALLEST`,
   `IGNITION_FILE` vs `IGNITION_CONFIG`).
4. `internal/run`: `os/exec` helper — stream stdout to `os.Stdout` (or file/tee),
   capture stderr, wrap non-zero exit with the stderr text for logging.
5. `internal/ignition`: port the current butane logic (validate-only, tee,
   stdout-only, file output).
6. `internal/install`: build the `flatcar-install` arg slice from Config; write inline
   ignition to a temp file; `DRY_RUN` -> `-y`.
7. `internal/action`: ff.Command handlers; `all` mode chains ignition -> install.
8. `main.go`: build the ff.Command tree; positional subcommand or `ACTION` env
   fallback; run; map errors to exit codes; unknown/no command -> `ffhelp` usage + exit 2.
9. Table-driven tests for config parsing + install arg assembly + dispatch.
10. `Dockerfile` multi-stage: `golang` builder (`CGO_ENABLED=0` static) -> butane base
    with pinned `flatcar-install` + `dnf` toolset + `COPY` binary;
    `ENTRYPOINT ["/flatcar-kit"]`.
11. Add an example Tinkerbell workflow template snippet (chained actions + volumes/pid).

---

## Tinkerbell action notes

- Config only via env (Tinkerbell `environment` map); structured JSON logs to stdout.
- `install` action: **privileged**, volumes `/dev:/dev`, plus a shared workdir volume for
  `ignition.json`; possibly `pid: host` for `udevadm` / partition-reread depending on the
  tink-worker runtime.
- Exit 0 success / non-zero fail.

---

## Verification

- `go build ./...` ; `go vet ./...` ; `go test ./...` (table-driven config + arg-assembly).
- `gofmt` / `golangci-lint` clean.
- ignition: `docker run -e ACTION=ignition -e BUTANE_CONFIG=... -v work:/work ...` -> `ignition.json`.
- install dry-run: `docker run --privileged -v /dev:/dev -e ACTION=install -e DEVICE=/dev/sdX
  -e DRY_RUN=true ...` -> `flatcar-install -y` prints settings, exit 0 (no disk writes).
- combined dry-run: `ACTION=all` with `BUTANE_CONFIG` + `DEVICE` + `DRY_RUN`.

---

## Open considerations (decide at implementation time)

- **`dnf` package set**: confirm which of the `flatcar-install` toolset is already in the
  butane base (Fedora) and install only the missing ones. Candidates: `gawk gnupg2 lvm2
  btrfs-progs util-linux e2fsprogs bzip2`(or `lbzip2`)`efibootmgr systemd-udev`.
  (A container check to confirm was not run.) Choose superset (simpler) vs minimized.
- **Pinned ref**: pick a `flatcar/init` commit SHA for `flatcar-install`.
- **Combined mode output path**: default `/tmp/ignition.json` handed to install.
- **`go.mod` Go version**: set a conservative minimum; do NOT bump it to match the build
  toolchain (the directive is the module's minimum required version).
- **ff env prefix**: currently no prefix (preserve existing names). Revisit if a
  `FLATCAR_` prefix is desired.

---

## Reference: existing behavior to port (from ignition-gen entrypoint.sh)

The current Bash logs JSON via `jq`:
`{timestamp, level, message}` with a UTC RFC3339 timestamp.

Ignition mode branches (to reproduce in Go):
- Validate only: `butane --strict $EXTRA_ARGS` on the config; on failure log stderr, exit 1.
- `STDOUT_ONLY`: emit Ignition JSON to stdout only.
- `TEE_TO_STDOUT`: write to `TARGET_PATH/TARGET_FILE` AND stdout.
- default: write to `TARGET_PATH/TARGET_FILE`.
- In all cases, capture butane stderr separately and log it on failure (do not merge
  stderr into the JSON output).
