#!/bin/bash
set -e

if command -v code &> /dev/null; then
    echo "Installing VS Code extensions..."
    
    extensions=(
        "Catppuccin.catppuccin-vsc"
        "PKief.material-icon-theme"
        "vscodevim.vim"
        "eamodio.gitlens"
        "ms-vscode.live-server"
        "esbenp.prettier-vscode"
        "dbaeumer.vscode-eslint"
        "ms-python.python"
        "ms-python.vscode-pylance"
        "golang.go"
        "rust-lang.rust-analyzer"
        "tamasfe.even-better-toml"
        "bradlc.vscode-tailwindcss"
        "github.copilot"
        "github.copilot-chat"
    )
    
    for extension in "${extensions[@]}"; do
        code --install-extension "$extension" --force
    done
fi
