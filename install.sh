#!/usr/bin/env bash
set -eu

REPO="DriesVanHool/lazyports"
BIN_NAME="lazyports"

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    printf 'error: required command not found: %s\n' "$1" >&2
    exit 1
  }
}

detect_os() {
  case "$(uname -s)" in
    Linux) printf 'linux' ;;
    Darwin) printf 'darwin' ;;
    *)
      printf 'error: unsupported OS for install.sh\n' >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) printf 'amd64' ;;
    arm64|aarch64) printf 'arm64' ;;
    *)
      printf 'error: unsupported architecture\n' >&2
      exit 1
      ;;
  esac
}

resolve_version() {
  if [ "${LAZYPORTS_VERSION:-}" != "" ]; then
    printf '%s' "$LAZYPORTS_VERSION"
    return
  fi

  curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | \
    grep '"tag_name":' | head -n1 | sed -E 's/.*"([^"]+)".*/\1/'
}

install_dir() {
  if [ -w /usr/local/bin ]; then
    printf '/usr/local/bin'
  else
    printf '%s/.local/bin' "$HOME"
  fi
}

main() {
  need_cmd curl
  need_cmd tar
  need_cmd install
  need_cmd mktemp

  os="$(detect_os)"
  arch="$(detect_arch)"
  version="$(resolve_version)"

  case "${os}-${arch}" in
    linux-amd64) archive="${BIN_NAME}-linux-amd64.tar.gz" ;;
    darwin-amd64) archive="${BIN_NAME}-darwin-amd64.tar.gz" ;;
    darwin-arm64) archive="${BIN_NAME}-darwin-arm64.tar.gz" ;;
    *)
      printf 'error: no release artifact for %s-%s\n' "$os" "$arch" >&2
      exit 1
      ;;
  esac

  url="https://github.com/${REPO}/releases/download/${version}/${archive}"
  tmpdir="$(mktemp -d)"
  trap 'rm -rf "$tmpdir"' EXIT INT TERM

  printf 'Installing %s %s for %s-%s\n' "$BIN_NAME" "$version" "$os" "$arch"
  curl -fsSL "$url" -o "$tmpdir/$archive"
  tar -xzf "$tmpdir/$archive" -C "$tmpdir"

  dest_dir="$(install_dir)"
  mkdir -p "$dest_dir"

  binary_path="$tmpdir/${BIN_NAME}-${os}-${arch}"
  if [ ! -f "$binary_path" ]; then
    printf 'error: binary not found in archive\n' >&2
    exit 1
  fi

  if [ -w "$dest_dir" ]; then
    install -m 0755 "$binary_path" "$dest_dir/$BIN_NAME"
  else
    need_cmd sudo
    sudo mkdir -p "$dest_dir"
    sudo install -m 0755 "$binary_path" "$dest_dir/$BIN_NAME"
  fi

  printf 'Installed to %s/%s\n' "$dest_dir" "$BIN_NAME"
  case ":$PATH:" in
    *":$dest_dir:"*) ;;
    *) printf 'Add %s to your PATH if it is not already there.\n' "$dest_dir" ;;
  esac
}

main "$@"
