#!/usr/bin/env bash
set -euo pipefail

repo="${1:-ayushjhaj8-sys/HumanSpeak}"
install_dir="${HUMANSPEAK_INSTALL_DIR:-$HOME/.local/bin}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"

case "$os" in
  darwin)
    if [[ "$arch" == "arm64" ]]; then
      asset="humanspeak-macos-arm64.tar.gz"
    else
      asset="humanspeak-macos-x64.tar.gz"
    fi
    ;;
  linux)
    if [[ "$arch" == "aarch64" || "$arch" == "arm64" ]]; then
      asset="humanspeak-linux-arm64.tar.gz"
    else
      asset="humanspeak-linux-x64.tar.gz"
    fi
    ;;
  *)
    echo "Unsupported OS: $os"
    exit 1
    ;;
esac

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

mkdir -p "$install_dir"
archive="$tmpdir/$asset"
url="https://github.com/$repo/releases/latest/download/$asset"

if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$url" -o "$archive"
else
  wget -q "$url" -O "$archive"
fi

tar -xzf "$archive" -C "$tmpdir"

if [[ "$os" == "darwin" || "$os" == "linux" ]]; then
  bin_name="humanspeak"
else
  bin_name="humanspeak"
fi

if [[ ! -f "$tmpdir/$bin_name" ]]; then
  if [[ -f "$tmpdir/HumanSpeak/$bin_name" ]]; then
    mv "$tmpdir/HumanSpeak/$bin_name" "$tmpdir/$bin_name"
  fi
fi

if [[ ! -f "$tmpdir/$bin_name" ]]; then
  echo "Installation failed: humanspeak binary not found in release archive."
  exit 1
fi

install_path="$install_dir/$bin_name"
install -m 755 "$tmpdir/$bin_name" "$install_path"

shell_profile=""
case "${SHELL:-}" in
  */zsh) shell_profile="$HOME/.zshrc" ;;
  */bash) shell_profile="$HOME/.bashrc" ;;
esac

if [[ -n "$shell_profile" ]]; then
  touch "$shell_profile"
  if ! grep -Fq "$install_dir" "$shell_profile"; then
    echo "export PATH=\"$install_dir:\$PATH\"" >> "$shell_profile"
  fi
fi

echo "HumanSpeak installed to $install_path"
echo "Restart your terminal, then run: humanspeak version"
