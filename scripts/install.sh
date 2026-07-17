#!/bin/sh
# GoScouter installer for Linux and macOS.
#
#   curl -fsSL https://raw.githubusercontent.com/GoScouter/GoScouter/main/scripts/install.sh | sh
#
# Options (flag or environment variable):
#   --version <tag>   GS_VERSION       release tag to install (default: latest)
#   --dir <path>      GS_INSTALL_DIR   install directory (default: /usr/local/bin
#                                      when writable, else ~/.local/bin)
#   --no-verify       GS_NO_VERIFY=1   skip the sha256 checksum check

set -eu

REPO="GoScouter/GoScouter"
BINARY="gs"

VERSION="${GS_VERSION:-}"
INSTALL_DIR="${GS_INSTALL_DIR:-}"
NO_VERIFY="${GS_NO_VERIFY:-}"

TMPDIR_GS=""

info() { printf '  %s\n' "$*"; }
warn() { printf '  warning: %s\n' "$*" >&2; }
die() {
	printf '  error: %s\n' "$*" >&2
	exit 1
}

cleanup() {
	# Preserve the caller's exit status: a trap whose last command fails would
	# otherwise become the script's status in dash/bash.
	status=$?
	if [ -n "$TMPDIR_GS" ]; then
		rm -rf "$TMPDIR_GS"
	fi
	return $status
}
trap cleanup EXIT INT TERM

while [ $# -gt 0 ]; do
	case "$1" in
	--version)
		VERSION="${2:-}"
		shift 2 || die "--version requires an argument"
		;;
	--dir)
		INSTALL_DIR="${2:-}"
		shift 2 || die "--dir requires an argument"
		;;
	--no-verify)
		NO_VERIFY=1
		shift
		;;
	-h | --help)
		cat <<-EOF
			GoScouter installer for Linux and macOS.

			Usage: install.sh [options]

			Options (flag or environment variable):
			  --version <tag>   GS_VERSION       release tag to install (default: latest)
			  --dir <path>      GS_INSTALL_DIR   install directory (default: /usr/local/bin
			                                     when writable, else ~/.local/bin)
			  --no-verify       GS_NO_VERIFY=1   skip the sha256 checksum check
			  -h, --help                         show this help
		EOF
		exit 0
		;;
	*) die "unknown option: $1 (try --help)" ;;
	esac
done

if command -v curl >/dev/null 2>&1; then
	http_get() { curl -fsSL "$1"; }
	http_download() { curl -fsSL --progress-bar -o "$2" "$1"; }
elif command -v wget >/dev/null 2>&1; then
	http_get() { wget -qO- "$1"; }
	http_download() { wget -q --show-progress -O "$2" "$1"; }
else
	die "neither curl nor wget found; install one and re-run"
fi

detect_os() {
	os="$(uname -s)"
	case "$os" in
	Linux) echo linux ;;
	Darwin) echo darwin ;;
	*) die "unsupported operating system: $os (this installer covers Linux and macOS; Windows users: see scripts/install.ps1)" ;;
	esac
}

detect_arch() {
	arch="$(uname -m)"
	case "$arch" in
	x86_64 | amd64) echo amd64 ;;
	aarch64 | arm64) echo arm64 ;;
	*) die "unsupported architecture: $arch" ;;
	esac
}

OS="$(detect_os)"
ARCH="$(detect_arch)"

if [ "$OS" = darwin ] && [ "$ARCH" = amd64 ]; then
	if [ "$(sysctl -n sysctl.proc_translated 2>/dev/null || echo 0)" = "1" ]; then
		ARCH=arm64
	fi
fi

ASSET="${BINARY}-${OS}-${ARCH}"

if [ -z "$VERSION" ]; then
	info "Resolving latest release..."
	VERSION="$(
		http_get "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null |
			sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' |
			head -n 1
	)" || true
	[ -n "$VERSION" ] || die "could not determine the latest release.
  The repository may not have published one yet — see
  https://github.com/${REPO}/releases
  You can build from source instead (needs Go and make):
    git clone https://github.com/${REPO}.git && cd GoScouter && make build"
fi

BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"

if [ -z "$INSTALL_DIR" ]; then
	if [ -w /usr/local/bin ] 2>/dev/null; then
		INSTALL_DIR=/usr/local/bin
	else
		INSTALL_DIR="${HOME}/.local/bin"
	fi
fi

mkdir -p "$INSTALL_DIR" || die "cannot create install directory: $INSTALL_DIR"
[ -w "$INSTALL_DIR" ] || die "install directory is not writable: $INSTALL_DIR
  Re-run with --dir <path>, or use sudo."

TMPDIR_GS="$(mktemp -d 2>/dev/null || mktemp -d -t gs-install)"

info "Installing GoScouter ${VERSION} (${OS}/${ARCH}) to ${INSTALL_DIR}"
http_download "${BASE_URL}/${ASSET}" "${TMPDIR_GS}/${ASSET}" ||
	die "download failed: ${BASE_URL}/${ASSET}
  Check that ${VERSION} exists and publishes a ${OS}/${ARCH} build."

if [ "$NO_VERIFY" = "1" ]; then
	warn "skipping checksum verification (--no-verify)"
else
	if command -v sha256sum >/dev/null 2>&1; then
		sha256_of() { sha256sum "$1" | cut -d' ' -f1; }
	elif command -v shasum >/dev/null 2>&1; then
		sha256_of() { shasum -a 256 "$1" | cut -d' ' -f1; }
	else
		sha256_of() { echo ""; }
	fi

	expected="$(
		http_get "${BASE_URL}/checksums.txt" 2>/dev/null |
			awk -v a="$ASSET" '$2 == a || $2 == "*" a { print $1; exit }'
	)" || true
	actual="$(sha256_of "${TMPDIR_GS}/${ASSET}")"

	if [ -z "$expected" ]; then
		warn "no checksum published for ${ASSET}; skipping verification"
	elif [ -z "$actual" ]; then
		warn "no sha256 tool found (sha256sum/shasum); skipping verification"
	elif [ "$expected" != "$actual" ]; then
		die "checksum mismatch for ${ASSET}
  expected: ${expected}
  actual:   ${actual}
  The download may be corrupt or tampered with — not installing."
	else
		info "Checksum verified."
	fi
fi

chmod +x "${TMPDIR_GS}/${ASSET}"

staged="${INSTALL_DIR}/.${BINARY}.tmp.$$"
mv -f "${TMPDIR_GS}/${ASSET}" "$staged" 2>/dev/null ||
	cp -f "${TMPDIR_GS}/${ASSET}" "$staged" ||
	die "failed to stage the binary in ${INSTALL_DIR}"
mv -f "$staged" "${INSTALL_DIR}/${BINARY}" || {
	rm -f "$staged"
	die "failed to install to ${INSTALL_DIR}/${BINARY}"
}

# macOS quarantines downloads; clear it so Gatekeeper doesn't block the binary.
if [ "$OS" = darwin ] && command -v xattr >/dev/null 2>&1; then
	xattr -d com.apple.quarantine "${INSTALL_DIR}/${BINARY}" 2>/dev/null || true
fi

info "Installed ${INSTALL_DIR}/${BINARY}"

case ":${PATH}:" in
*":${INSTALL_DIR}:"*) ;;
*)
	warn "${INSTALL_DIR} is not on your PATH."
	info "Add it to your shell profile, e.g.:"
	info "  echo 'export PATH=\"${INSTALL_DIR}:\$PATH\"' >> ~/.profile"
	;;
esac

info "Run '${BINARY} --help' to get started."
