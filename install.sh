#!/usr/bin/env bash
set -eu

umask 022

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

  latest_url="$(curl --proto '=https' --tlsv1.2 -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/${REPO}/releases/latest")"
  version="${latest_url##*/}"
  if [ -z "$version" ] || [ "$version" = "latest" ]; then
    printf 'error: could not resolve the latest release for %s\n' "$REPO" >&2
    exit 1
  fi
  printf '%s' "$version"
}

install_dir() {
  if [ "${LAZYPORTS_INSTALL_DIR:-}" != "" ]; then
    printf '%s' "$LAZYPORTS_INSTALL_DIR"
    return
  fi

  if [ -w /usr/local/bin ]; then
    printf '/usr/local/bin'
  else
    printf '%s/.local/bin' "$HOME"
  fi
}

download() {
  local url="$1"
  local output="$2"
  curl --proto '=https' --tlsv1.2 -fL --retry 3 --retry-delay 1 --connect-timeout 10 "$url" -o "$output"
}

hash_file() {
  file="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file"
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file"
    return
  fi

  printf 'error: sha256sum or shasum is required for checksum verification\n' >&2
  exit 1
}

verify_checksum() {
  archive="$1"
  checksums_file="$2"
  archive_name="$(basename "$archive")"
  expected_checksum=""

  while IFS= read -r line; do
    case "$line" in
      *"  ${archive_name}")
        expected_checksum="${line%% *}"
        break
        ;;
    esac
  done < "$checksums_file"

  if [ -z "$expected_checksum" ]; then
    printf 'error: checksum entry not found for %s\n' "${archive##*/}" >&2
    exit 1
  fi

  actual_checksum="$(hash_file "$archive")"
  actual_checksum="${actual_checksum%% *}"

  if [ "$expected_checksum" != "$actual_checksum" ]; then
    printf 'error: checksum verification failed for %s\n' "${archive##*/}" >&2
    exit 1
  fi
}

main() {
  need_cmd curl
  need_cmd tar
  need_cmd install
  need_cmd mktemp
  need_cmd grep

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
  checksums_url="https://github.com/${REPO}/releases/download/${version}/checksums.txt"
  tmpdir="$(mktemp -d)"
  trap 'rm -rf "$tmpdir"' EXIT INT TERM

  printf 'Installing %s %s for %s-%s\n' "$BIN_NAME" "$version" "$os" "$arch"
  download "$checksums_url" "$tmpdir/checksums.txt"
  download "$url" "$tmpdir/$archive"
  verify_checksum "$tmpdir/$archive" "$tmpdir/checksums.txt"
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
