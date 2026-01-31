#!/bin/bash
set -e

echo "Installing Bitwarden CLI..."

if command -v bw &> /dev/null; then
    echo "Bitwarden CLI already installed"
    exit 0
fi

{{ if eq .chezmoi.os "darwin" }}
brew install bitwarden-cli
{{ else if eq .chezmoi.os "linux" }}
if command -v snap &> /dev/null; then
    sudo snap install bw
else
    # Download binary
    curl -Lo bw.zip "https://vault.bitwarden.com/download/?app=cli&platform=linux"
    unzip bw.zip
    chmod +x bw
    sudo mv bw /usr/local/bin/
    rm bw.zip
fi
{{ end }}

echo "Bitwarden CLI installed. Please run 'bw login' to authenticate."
