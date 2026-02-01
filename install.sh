#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}================================${NC}"
echo -e "${GREEN}  Dotfiles Installation Script${NC}"
echo -e "${GREEN}================================${NC}"
echo ""

# Detect OS
OS="$(uname -s)"
case "${OS}" in
    Linux*)     MACHINE=Linux;;
    Darwin*)    MACHINE=Mac;;
    *)          MACHINE="UNKNOWN:${OS}"
esac

echo -e "${YELLOW}Detected OS: ${MACHINE}${NC}"
echo ""

# Install Homebrew on macOS
if [ "$MACHINE" = "Mac" ]; then
    if ! command -v brew &> /dev/null; then
        echo -e "${YELLOW}Installing Homebrew...${NC}"
        /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
    else
        echo -e "${GREEN}Homebrew already installed${NC}"
    fi
fi

# Install chezmoi if not present
if ! command -v chezmoi &> /dev/null; then
    echo -e "${YELLOW}Installing chezmoi...${NC}"
    if [ "$MACHINE" = "Mac" ]; then
        brew install chezmoi
    else
        sh -c "$(curl -fsLS get.chezmoi.io)" -- -b "$HOME/.local/bin"
        export PATH="$HOME/.local/bin:$PATH"
    fi
else
    echo -e "${GREEN}chezmoi already installed${NC}"
fi

# Install Bitwarden CLI if not present
if ! command -v bw &> /dev/null; then
    echo -e "${YELLOW}Installing Bitwarden CLI...${NC}"
    if [ "$MACHINE" = "Mac" ]; then
        brew install bitwarden-cli
    else
        # For Linux, install via npm or snap
        if command -v npm &> /dev/null; then
            npm install -g @bitwarden/cli
        elif command -v snap &> /dev/null; then
            sudo snap install bw
        else
            echo -e "${RED}Please install Bitwarden CLI manually from: https://bitwarden.com/help/cli/${NC}"
            exit 1
        fi
    fi
else
    echo -e "${GREEN}Bitwarden CLI already installed${NC}"
fi

echo ""
echo -e "${YELLOW}Checking Bitwarden authentication...${NC}"

# Check if Bitwarden is logged in and unlocked
if ! bw login --check &> /dev/null; then
    echo -e "${RED}Bitwarden is not logged in.${NC}"
    echo -e "${YELLOW}Please run:${NC}"
    echo "  bw login"
    echo "  export BW_SESSION=\"\$(bw unlock --raw)\""
    echo ""
    echo -e "${YELLOW}Then re-run this script.${NC}"
    exit 1
fi

if [ -z "$BW_SESSION" ]; then
    echo -e "${RED}Bitwarden session not found in environment.${NC}"
    echo -e "${YELLOW}Please run:${NC}"
    echo "  export BW_SESSION=\"\$(bw unlock --raw)\""
    echo ""
    echo -e "${YELLOW}Then re-run this script.${NC}"
    exit 1
fi

echo -e "${GREEN}Bitwarden is authenticated and unlocked${NC}"
echo ""

# Initialize chezmoi
REPO_URL="https://github.com/CuriousFurBytes/.dotfiles.git"
echo -e "${YELLOW}Initializing chezmoi with dotfiles repository...${NC}"

if [ -d "$HOME/.local/share/chezmoi/.git" ]; then
    echo -e "${GREEN}Dotfiles already initialized${NC}"
else
    chezmoi init "$REPO_URL"
fi

# Apply dotfiles
echo ""
echo -e "${YELLOW}Applying dotfiles...${NC}"
chezmoi apply -v

echo ""
echo -e "${GREEN}================================${NC}"
echo -e "${GREEN}  Installation Complete!${NC}"
echo -e "${GREEN}================================${NC}"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "  1. Restart your shell or run: exec \$SHELL"
echo "  2. Verify installations with: git config --list"
echo "  3. Check SSH key with: ssh-add -l"
echo ""
echo -e "${GREEN}Enjoy your new dotfiles! 🎉${NC}"
