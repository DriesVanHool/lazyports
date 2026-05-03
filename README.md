# LazyPorts

LazyPorts is a chill little terminal app for finding what is hogging your ports and politely showing it the door.

It gives you a fast TUI for browsing port bindings plus a small CLI for listing and killing by port.

## Features

- TUI for browsing active port bindings
- Fuzzy search with `/`
- Kill the selected process with `k`
- CLI commands for `list`, `kill`, and `version`
- Cross-platform support for Linux, macOS, and Windows

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

## Quick Start

```bash
lazyports
lazyports list
lazyports kill 8080
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

Kill every process bound to a port:

```bash
lazyports kill 8080
```

Run from source:

```bash
go run .
go run . list
go run . kill 8080
go run . version
```

Install with Go:

```bash
go install github.com/DriesVanHool/lazyports@latest
```

## TUI Keys

- `/` start fuzzy search
- `r` refresh the port list
- `k` kill the selected process
- `Enter` show details for the selected row
- `q` quit

## Platform Notes

- Linux: uses `lsof`, with `ss` as a fallback
- macOS: uses `lsof`
- Windows: uses `netstat` and `tasklist`

Some port listings and kill actions may require elevated privileges depending on the target process and OS.

## Build

```bash
go build ./...
make cross-build
make package-release VERSION=v0.1.0
```

## Release Artifacts

Each tagged release publishes:

- `lazyports-linux-amd64.tar.gz`
- `lazyports-darwin-amd64.tar.gz`
- `lazyports-darwin-arm64.tar.gz`
- `lazyports-windows-amd64.zip`
- `checksums.txt`
