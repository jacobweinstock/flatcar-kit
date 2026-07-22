# syntax=docker/dockerfile:1

# BUILD_IN_CONTAINER controls where the Go binary comes from.
#   false (default): copy a binary that was built outside the container
#                    (on the host or in CI) from out/flatcar-kit-${TARGETOS}-${TARGETARCH}.
#   true:            compile the binary inside the container (needs the Go toolchain).
ARG BUILD_IN_CONTAINER=false
ARG TARGETOS
ARG TARGETARCH

# ---- build stage: compile inside the container -----------------------------
FROM golang:1.26 AS build-true

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static, stripped binary so it runs on the butane (Fedora) base without libc deps.
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
        go build -trimpath -ldflags="-s -w" -o /flatcar-kit .

# ---- build stage: use a prebuilt binary from the build context -------------
FROM scratch AS build-false

ARG TARGETOS
ARG TARGETARCH
COPY out/flatcar-kit-${TARGETOS}-${TARGETARCH} /flatcar-kit

# ---- select the build stage based on BUILD_IN_CONTAINER --------------------
FROM build-${BUILD_IN_CONTAINER} AS build

# ---- runtime stage ---------------------------------------------------------
FROM quay.io/coreos/butane:latest

# Pin flatcar-install to a specific flatcar/init commit rather than a moving branch.
ARG FLATCAR_INIT_REF=1b9b634d69cec244367fd70c6bc2ce8e4239bdf2

# Toolset required by flatcar-install.
RUN dnf install -y --setopt=install_weak_deps=False --nodocs \
        gawk \
        gnupg2 \
        grep \
        sed \
        wget \
        util-linux \
        coreutils \
        lvm2 \
        btrfs-progs \
        bzip2 \
        systemd-udev \
        efibootmgr \
    && dnf clean all \
    && rm -rf /var/cache/dnf

RUN wget -O /usr/local/bin/flatcar-install \
        "https://raw.githubusercontent.com/flatcar/init/${FLATCAR_INIT_REF}/bin/flatcar-install" \
    && chmod +x /usr/local/bin/flatcar-install

COPY --from=build /flatcar-kit /flatcar-kit

ENTRYPOINT ["/flatcar-kit"]
