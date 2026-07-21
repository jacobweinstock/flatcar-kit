# flatcar-kit

`ghcr.io/jacobweinstock/flatcar-kit:latest`

A [Tinkerbell](https://github.com/tinkerbell) Action for provisioning Flatcar
Container Linux. It has three modes:

1. `ignition` — transpile a [Butane](https://coreos.github.io/butane/) config into an Ignition config.
2. `install` — write Flatcar to a disk.
3. `all` — transpile a Butane config and install Flatcar in a single action.

Configuration is provided entirely through the Action `environment` map, logs
are emitted as structured JSON to stdout, and the Action succeeds (exit `0`) or
fails (non-zero) like any other Tinkerbell Action.

## Mode selection

The mode is chosen by passing the subcommand as the Action `command`:

```yaml
command: ["ignition"]   # or: ["install"] | ["all"]
```

## Environment variables

All configuration is provided as `UPPER_SNAKE_CASE` environment variables in the
Action `environment` map.

### `ignition` mode

| Name | Type | Default | Required | Description |
| --- | --- | --- | --- | --- |
| `BUTANE_CONFIG` | string | `""` | yes | Butane config document to transpile. |
| `TARGET_PATH` | string | `/tmp/` | no | Output directory for the Ignition file. |
| `TARGET_FILE` | string | `ignition.json` | no | Output file name for the Ignition file. |
| `VALIDATE_ONLY` | bool | `false` | no | Only validate the Butane config; do not transpile. |
| `TEE_TO_STDOUT` | bool | `false` | no | Write the Ignition output to the target file **and** stdout. |
| `STDOUT_ONLY` | bool | `false` | no | Write the Ignition output to stdout only. |
| `EXTRA_ARGS` | string | `""` | no | Additional arguments passed to `butane`. |

`TEE_TO_STDOUT` and `STDOUT_ONLY` are mutually exclusive.

### `install` mode

| Name | Type | Default | Required | Description |
| --- | --- | --- | --- | --- |
| `DEVICE` | string | `""` | yes\* | Target block device, e.g. `/dev/sda`. |
| `INSTALL_TO_SMALLEST` | bool | `false` | yes\* | Install to the smallest available disk. |
| `IGNITION_FILE` | string | `""` | no | Path to an existing Ignition config file. |
| `IGNITION_CONFIG` | string | `""` | no | Inline Ignition config. |
| `CHANNEL` | string | `""` | no | Flatcar release channel. |
| `VERSION` | string | `""` | no | Flatcar version. |
| `BOARD` | string | `""` | no | Target board. |
| `OEM` | string | `""` | no | OEM id. |
| `BASE_URL` | string | `""` | no | Base URL for image download. |
| `KEYFILE` | string | `""` | no | GPG key file for signature verification. |
| `IMAGE_FILE` | string | `""` | no | Local image file to install. |
| `COPY_NET` | bool | `false` | no | Copy network units to the installed system. |
| `CREATE_UEFI` | bool | `false` | no | Create a UEFI boot entry. |
| `DRY_RUN` | bool | `false` | no | Print the install settings without writing to disk. |
| `DOWNLOAD_ONLY` | bool | `false` | no | Download the image only; do not install. |
| `EXTRA_ARGS` | string | `""` | no | Additional arguments passed through to the installer. |

\* Exactly one of `DEVICE` or `INSTALL_TO_SMALLEST` is required; they are
mutually exclusive. `IGNITION_FILE` and `IGNITION_CONFIG` are also mutually
exclusive.

### `all` mode

`all` accepts the union of the `ignition` and `install` variables. The Butane
config is transpiled and its Ignition output is installed within the same
action, so no shared volume is needed for the Ignition output. In this mode
`EXTRA_ARGS` applies to the Butane transpile step.

## Tinkerbell Action

The following Action fields matter for `flatcar-kit`:

| Field | Description | Required |
| --- | --- | --- |
| `image` | `ghcr.io/jacobweinstock/flatcar-kit:latest` or a pinned tag. | Yes |
| `command` | The mode subcommand: `["ignition"]`, `["install"]`, or `["all"]`. | Yes |
| `environment` | The mode variables, detailed above. | Yes |
| `volumes` | `install`/`all` need `/dev:/dev` for raw block-device access; `ignition` output consumed by a later action needs a shared workdir volume. | For `install`/`all` |

## Usage

The Hardware fields below are resolved by Tinkerbell at render time from the
Hardware's spec:

- disk device — `{{ (index .hardware.spec.disks 0).device }}`
- SSH keys — `{{ range .hardware.spec.metadata.instance.ssh_keys }}`

### `ignition` — transpile Butane, then install from the file

Two actions on a shared `/out` volume: the first writes `ignition.json`, the
second installs Flatcar and consumes it.

```yaml
volumes:
  - /tmp/tink:/out
  - /dev:/dev
actions:
  - name: "butane-to-ignition"
    image: ghcr.io/jacobweinstock/flatcar-kit:latest
    timeout: 120
    command: ["ignition"]
    environment:
      TARGET_PATH: /out
      TARGET_FILE: ignition.json
      BUTANE_CONFIG: |
        variant: flatcar
        version: 1.0.0
        passwd:
          users:
            - name: core
              ssh_authorized_keys:
                {{- range .hardware.spec.metadata.instance.ssh_keys }}
                - {{ . }}
                {{- end }}
  - name: "flatcar-install"
    image: ghcr.io/jacobweinstock/flatcar-kit:latest
    timeout: 1500
    pid: host
    command: ["install"]
    environment:
      DEVICE: "{{ (index .hardware.spec.disks 0).device }}"
      CHANNEL: stable
      VERSION: current
      IGNITION_FILE: /out/ignition.json
```

### `install` — install from an inline Ignition config

A single action that installs Flatcar using a pre-rendered Ignition config
provided inline (use `IGNITION_FILE` instead to point at a mounted file).

```yaml
volumes:
  - /dev:/dev
actions:
  - name: "flatcar-install"
    image: ghcr.io/jacobweinstock/flatcar-kit:latest
    timeout: 1500
    pid: host
    command: ["install"]
    environment:
      DEVICE: "{{ (index .hardware.spec.disks 0).device }}"
      CHANNEL: stable
      VERSION: current
      IGNITION_CONFIG: |
        {
          "ignition": { "version": "3.4.0" },
          "passwd": {
            "users": [
              {
                "name": "core",
                "sshAuthorizedKeys": [
                  {{- range $i, $k := .hardware.spec.metadata.instance.ssh_keys }}
                  {{- if $i }},{{ end }}
                  "{{ $k }}"
                  {{- end }}
                ]
              }
            ]
          }
        }
```

### `all` — transpile and install in one action

The Butane config is transpiled and installed within the same action, so only
`/dev` needs to be mounted.

```yaml
volumes:
  - /dev:/dev
actions:
  - name: "butane-and-install"
    image: ghcr.io/jacobweinstock/flatcar-kit:latest
    timeout: 1500
    pid: host
    command: ["all"]
    environment:
      DEVICE: "{{ (index .hardware.spec.disks 0).device }}"
      CHANNEL: stable
      VERSION: current
      BUTANE_CONFIG: |
        variant: flatcar
        version: 1.0.0
        passwd:
          users:
            - name: core
              ssh_authorized_keys:
                {{- range .hardware.spec.metadata.instance.ssh_keys }}
                - {{ . }}
                {{- end }}
```

Complete, runnable Templates for each mode are in [`examples/`](examples/):

- [`examples/template-ignition.yaml`](examples/template-ignition.yaml)
- [`examples/template-install.yaml`](examples/template-install.yaml)
- [`examples/template-all.yaml`](examples/template-all.yaml)

## Building

```sh
docker build -t flatcar-kit .
```
