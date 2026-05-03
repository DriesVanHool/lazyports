<p align="center">
  <img src="logo.png" alt="Logo" width="400"/>
</p>

**LazyPorts** is a terminal app for seeing what is using your ports and stopping it quickly.

It gives you a fast TUI for browsing port bindings and a simple CLI for listing or terminating by port.

<img src="screenshot.png" alt="LazyPorts screenshot" width="1000"/>

## Features

- TUI for browsing active port bindings
- Fuzzy search with `/`
- Graceful terminate first, with a force-kill fallback when needed
- Kill the selected process with `k`
- CLI commands for `list`, `kill`, and `version`
- Cross-platform support for Linux, macOS, and Windows
- No telemetry

## Install

Install the latest release on Linux or macOS:

```bash
curl -fsSL https://raw.githubusercontent.com/DriesVanHool/lazyports/main/install.sh | bash
```

Install a specific release:

```bash
LAZYPORTS_VERSION=v0.1.0 curl -fsSL https://raw.githubusercontent.com/DriesVanHool/lazyports/main/install.sh | bash
```

If `~/.local/bin` is not on your `PATH`, add it after install.

Prefer to inspect the installer first:

```bash
curl -fsSL https://raw.githubusercontent.com/DriesVanHool/lazyports/main/install.sh -o install.sh
less install.sh
bash install.sh
```

Install to a custom directory:

```bash
LAZYPORTS_INSTALL_DIR="$HOME/.local/bin" curl -fsSL https://raw.githubusercontent.com/DriesVanHool/lazyports/main/install.sh | bash
```

Install with Go:

```bash
go install github.com/DriesVanHool/lazyports@latest
```

## Quick Start

```bash
lazyports
lazyports list
lazyports list --all
lazyports kill 8080
lazyports kill 8080 --graceful-only
lazyports kill 8080 --force
lazyports version
```

## Usage

Launch the TUI:

```bash
lazyports
```

List active port bindings:

```bash
lazyports list
```

List listeners and active connections:

```bash
lazyports list --all
```

Terminate every listening process bound to a port. By default, LazyPorts tries a graceful stop first and only force kills if needed:

```bash
lazyports kill 8080
```

Only attempt graceful termination:

```bash
lazyports kill 8080 --graceful-only
```

Force kill immediately:

```bash
lazyports kill 8080 --force
```

## Run From Source

```bash
go run .
go run . list
go run . list --all
go run . kill 8080
go run . version
```

## Build

Build the local binary:

```bash
go build ./...
```

Create packaged release artifacts:

```bash
make cross-build
make package-release VERSION=v0.1.0
```

## Release Artifacts

Tagged releases publish platform builds such as:

- `lazyports-linux-amd64.tar.gz`
- `lazyports-darwin-amd64.tar.gz`
- `lazyports-darwin-arm64.tar.gz`
- `lazyports-windows-amd64.zip`
- `checksums.txt`

## TUI Keys

- `/` start fuzzy search
- `a` toggle between listeners and all connections
- `r` refresh the port list
- `k` terminate the selected process
- `Enter` show details for the selected row
- `q` quit

## Platform Notes

- Linux: uses `lsof`, with `ss` as a fallback
- macOS: uses `lsof`
- Windows: uses `netstat` and `tasklist`

Some listings and termination actions may require elevated privileges depending on the target process and OS.
