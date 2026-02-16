#!/bin/bash
set -e

# ── Colors ──────────────────────────────────────────────────────────
BOLD='\033[1m'
BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
DIM='\033[2m'
RESET='\033[0m'

info()  { echo -e "  ${BLUE}[info]${RESET} $1"; }
ok()    { echo -e "  ${GREEN}[ok]${RESET} $1"; }
done_() { echo -e "  ${BLUE}[done]${RESET} $1"; }
skip()  { echo -e "  ${YELLOW}[skip]${RESET} $1"; }
fail()  { echo -e "  ${RED}[fail]${RESET} $1"; }

header() {
    echo ""
    echo -e "${DIM}─────────────────────────────────────────────────────────────${RESET}"
    echo -e "  ${BOLD}$1${RESET}"
    echo -e "${DIM}─────────────────────────────────────────────────────────────${RESET}"
    echo ""
}

confirm() {
    local msg="$1"
    read -rp "$(echo -e "  ${BOLD}$msg${RESET} [Y/n] ")" answer
    case "$answer" in
        [nN]*) return 1 ;;
        *) return 0 ;;
    esac
}

# ── Detect OS ───────────────────────────────────────────────────────
OS="$(uname -s)"

echo ""
echo -e "${BOLD}╔══════════════════════════════════════╗${RESET}"
echo -e "${BOLD}║   Dotfiles Bootstrap Installer       ║${RESET}"
echo -e "${BOLD}╚══════════════════════════════════════╝${RESET}"
echo ""
echo -e "  ${BOLD}OS:${RESET} $OS"
echo -e "  ${BOLD}User:${RESET} $(whoami)"
echo -e "  ${BOLD}Host:${RESET} $(hostname)"
echo ""

# ── Homebrew ────────────────────────────────────────────────────────
header "Homebrew"

if command -v brew &>/dev/null; then
    ok "Homebrew"
else
    if confirm "Install Homebrew?"; then
        info "Installing Homebrew..."
        /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

        # Add to PATH
        if [[ "$OS" == "Darwin" ]]; then
            if [[ $(uname -m) == "arm64" ]]; then
                eval "$(/opt/homebrew/bin/brew shellenv)"
            else
                eval "$(/usr/local/bin/brew shellenv)"
            fi
        else
            eval "$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)"
        fi
        done_ "Homebrew"
    else
        skip "Homebrew"
    fi
fi

# ── Zerobrew ────────────────────────────────────────────────────────
header "Zerobrew"

if command -v zb &>/dev/null; then
    ok "Zerobrew"
else
    if command -v brew &>/dev/null; then
        if confirm "Install Zerobrew?"; then
            info "Installing Zerobrew..."
            curl -fsSL https://zerobrew.rs/install | bash
            done_ "Zerobrew"
        else
            skip "Zerobrew"
        fi
    else
        skip "Zerobrew (Homebrew not installed)"
    fi
fi

# ── Go ──────────────────────────────────────────────────────────────
header "Go"

if command -v go &>/dev/null; then
    ok "Go ($(go version | awk '{print $3}'))"
else
    if confirm "Install Go?"; then
        info "Installing Go..."
        if [[ "$OS" == "Darwin" ]]; then
            brew install go
        else
            if command -v apt &>/dev/null; then
                sudo apt install -y golang-go
            elif command -v dnf &>/dev/null; then
                sudo dnf install -y golang
            else
                brew install go
            fi
        fi
        done_ "Go"
    else
        fail "Go is required to run the installer"
        exit 1
    fi
fi

# ── Snap (Linux only) ──────────────────────────────────────────────
if [[ "$OS" == "Linux" ]]; then
    header "Snap"

    if command -v snap &>/dev/null; then
        ok "Snap"
    else
        if confirm "Install Snap?"; then
            info "Installing Snap..."
            if command -v apt &>/dev/null; then
                sudo apt install -y snapd
            elif command -v dnf &>/dev/null; then
                sudo dnf install -y snapd
            fi
            done_ "Snap"
        else
            skip "Snap"
        fi
    fi
fi

# ── Cargo (Rust) ───────────────────────────────────────────────────
header "Cargo (Rust)"

if command -v cargo &>/dev/null; then
    ok "Cargo ($(cargo --version | awk '{print $2}'))"
else
    if confirm "Install Rust/Cargo via rustup?"; then
        info "Installing Rust via rustup..."
        curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
        source "$HOME/.cargo/env"
        done_ "Cargo"
    else
        skip "Cargo"
    fi
fi

# ── uv (Python) ────────────────────────────────────────────────────
header "uv (Python package manager)"

if command -v uv &>/dev/null; then
    ok "uv ($(uv --version 2>/dev/null | awk '{print $2}'))"
else
    if confirm "Install uv?"; then
        info "Installing uv..."
        if command -v brew &>/dev/null; then
            brew install uv
        else
            curl -LsSf https://astral.sh/uv/install.sh | sh
        fi
        done_ "uv"
    else
        skip "uv"
    fi
fi

# ── Build and run the Go installer ─────────────────────────────────
header "Building Installer TUI"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
INSTALLER_DIR="$SCRIPT_DIR/installer"
INSTALLER_BIN="/tmp/dotfiles-installer"

if [[ ! -d "$INSTALLER_DIR" ]]; then
    # If running from curl, clone the repo first
    info "Cloning dotfiles repository..."
    TEMP_DIR=$(mktemp -d)
    git clone https://github.com/CuriousFurBytes/.dotfiles.git "$TEMP_DIR"
    INSTALLER_DIR="$TEMP_DIR/installer"
    SCRIPT_DIR="$TEMP_DIR"
fi

info "Building installer..."
cd "$INSTALLER_DIR"
go build -o "$INSTALLER_BIN" .

if [[ $? -ne 0 ]]; then
    fail "Failed to build installer"
    exit 1
fi
done_ "Installer built"

echo ""
info "Launching installer TUI..."
echo ""

exec "$INSTALLER_BIN" --source "$SCRIPT_DIR"
