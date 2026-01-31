#!/bin/bash
set -e

# Install television on Linux
if ! command -v tv &> /dev/null; then
    echo "Installing television..."
    curl -fsSL https://alexpasmantier.github.io/television/install.sh | bash
else
    echo "✓ television already installed, skipping"
fi
