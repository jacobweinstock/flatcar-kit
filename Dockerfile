# syntax=docker/dockerfile:1

# ---- build stage -----------------------------------------------------------
FROM golang:1.26 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Static, stripped binary so it runs on the butane (Fedora) base without libc deps.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /flatcar-kit .

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
