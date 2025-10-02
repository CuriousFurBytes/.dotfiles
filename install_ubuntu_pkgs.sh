#!/bin/sh

sudo apt update

# ZSH
sudo apt install zsh

# Oh-my-zsh
sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"

# Oh-my-zsh plugins
git clone https://github.com/zsh-users/zsh-autosuggestions ${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/zsh-autosuggestions
git clone https://github.com/zsh-users/zsh-syntax-highlighting.git ${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/zsh-syntax-highlighting

# Pure
mkdir -p "$HOME/.zsh"
git clone https://github.com/sindresorhus/pure.git "$HOME/.zsh/pure"

# Homeshick
git clone https://github.com/andsens/homeshick.git $HOME/.homesick/repos/homeshick
source "$HOME/.homesick/repos/homeshick/homeshick.sh"

homeshick clone lucasqueiroz/.dotfiles

# Setup Nix
nix-channel --add https://nixos.org/channels/nixpkgs-unstable nixpkgs

nix-channel --update

# FZF
nix-env -iA nixpkgs.fzf

# EZA
nix-env -iA nixpkgs.eza

# Glow
nix-env -iA nixpkgs.glow

# Bottom
nix-env -iA nixpkgs.bottom

# Zoxide
nix-env -iA nixpkgs.zoxide

# Bat
nix-env -iA nixpkgs.bat

# Kitty
nix-env -iA nixpkgs.kitty
ln -sf ~/.local/kitty.app/bin/kitty ~/.local/kitty.app/bin/kitten ~/.local/bin/
cp ~/.local/kitty.app/share/applications/kitty.desktop ~/.local/share/applications/
sed -i "s|Icon=kitty|Icon=$(readlink -f ~)/.local/kitty.app/share/icons/hicolor/256x256/apps/kitty.png|g" ~/.local/share/applications/kitty*.desktop
sed -i "s|Exec=kitty|Exec=$(readlink -f ~)/.local/kitty.app/bin/kitty|g" ~/.local/share/applications/kitty*.desktop
echo 'kitty.desktop' > ~/.config/xdg-terminals.list

# Thunderbird
nix-env -iA nixpkgs.thunderbird

# Obsidian
nix-env -iA nixpkgs.obsidian

# VS Code
nix-env -iA nixpkgs.vscode

# Uv
nix-env -iA nixpkgs.uv

# Podman
nix-env -iA nixpkgs.podman

# Lazygit
nix-env -iA nixpkgs.lazygit

# Lazydocker
nix-env -iA nixpkgs.lazydocker

# Solaar
nix-env -iA nixpkgs.solaar

# NVim
nix-env -iA nixpkgs.neovim

# RClone
nix-env -iA nixpkgs.rclone

# ProtonVPN
wget https://repo.protonvpn.com/debian/dists/stable/main/binary-all/protonvpn-stable-release_1.0.8_all.deb
sudo dpkg -i ./protonvpn-stable-release_1.0.8_all.deb && sudo apt update
sudo apt install proton-vpn-gnome-desktop

# Bitwarden
sudo snap install bitwarden

