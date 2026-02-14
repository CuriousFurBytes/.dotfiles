#!/bin/bash
set -e

# Ensure python3 is available
if ! command -v python3 &>/dev/null; then
    OS="$(uname -s)"
    case "${OS}" in
        Darwin*)
            echo "Python 3 not found. Installing via Homebrew..."
            brew install python3
            ;;
        Linux*)
            if command -v apt &>/dev/null; then
                echo "Python 3 not found. Installing via apt..."
                sudo apt update && sudo apt install -y python3
            elif command -v dnf &>/dev/null; then
                echo "Python 3 not found. Installing via dnf..."
                sudo dnf install -y python3
            else
                echo "Error: Python 3 is required but could not be installed automatically."
                exit 1
            fi
            ;;
        *)
            echo "Error: Unsupported OS. Please install Python 3 manually."
            exit 1
            ;;
    esac
fi

# Download and run the bootstrap script
curl -fsSL https://raw.githubusercontent.com/CuriousFurBytes/.dotfiles/main/.install_bootstrap.py | python3 -B -
