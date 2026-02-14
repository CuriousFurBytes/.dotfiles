<div align="center">
  <img src="assets/header.svg" alt="Dotfiles Header" width="100%">
</div>

# Dotfiles

Personal dotfiles managed with [chezmoi](https://www.chezmoi.io/).

## What's Included

- **Zsh** configuration with [Pure](https://github.com/sindresorhus/pure) prompt
- **Git** configuration with signing support
- **SSH** configuration
- **Zen Browser** custom styles and preferences
- Common aliases
- Tool configurations (fzf, zoxide, nvm, etc.)

## Prerequisites

- [Proton Pass CLI](https://protonpass.github.io/pass-cli/) for secrets management
- macOS (uses Homebrew for package installation)

## Installation

### Quick Install

Run this single command to install everything:

```bash
curl -fsSL https://raw.githubusercontent.com/CuriousFurBytes/.dotfiles/main/install.sh | bash
```

This will automatically:
- Install Homebrew (macOS) or required package managers
- Install chezmoi
- Install Proton Pass CLI
- Clone and apply your dotfiles

**Prerequisites:**
- **Proton Pass CLI** must be authenticated before running the script:
  ```bash
  pass-cli login
  ```

- **Required Proton Pass Items** in your vault:

  | Item | Type | Description |
  |------|------|-------------|
  | SSH Key | SSH Key | Private key in key field, public key in Notes |
  | Git Signing Key | Login/Secure Note | GPG or signing key |

### Manual Installation

If you prefer to install manually:

1. **Install Homebrew** (macOS):
   ```bash
   /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
   ```

2. **Install Proton Pass CLI and authenticate**:
   ```bash
   # macOS
   brew tap protonpass/tap
   brew install protonpass/tap/pass-cli

   # Linux
   curl -fsSL https://proton.me/download/pass-cli/install.sh | bash

   # Authenticate
   pass-cli login
   ```

3. **Initialize and apply dotfiles**:
   ```bash
   chezmoi init https://github.com/CuriousFurBytes/.dotfiles.git
   chezmoi apply -v
   ```

The installation will:
- Install Homebrew packages from `packages.json`
- Set up Oh My Zsh with plugins
- Install Pure prompt
- Configure SSH keys and Git

### Post-Configuration

After applying chezmoi:

1. **Restart your shell** or source the configuration:
   ```bash
   exec zsh
   # or
   source ~/.zshrc
   ```

2. **Verify installations**:
   ```bash
   # Check zsh prompt
   echo $PROMPT
   
   # Check git config
   git config --list
   
   # Verify SSH key
   ssh-add -l
   ```

3. **Optional**: Set up Git signing if configured:
   ```bash
   git config --global commit.gpgsign true
   ```

4. **Zen Browser** (if installed):
   - Custom styles and preferences are automatically copied to your active profile
   - Extensions are auto-installed via `policies.json` (requires sudo)
   - Restart Zen Browser to apply changes

## Customization

### Zen Browser

Edit your Zen Browser configuration:

```bash
# Custom UI styles
chezmoi edit ~/.config/zen-browser/chrome/userChrome.css

# Browser preferences
chezmoi edit ~/.config/zen-browser/user.js

# Extension list
chezmoi edit ~/.config/zen-browser/extensions.json
```

**Adding extensions:**

1. Find the extension on [addons.mozilla.org](https://addons.mozilla.org)
2. Get the extension ID from `about:debugging` in Zen Browser
3. Add to `extensions.json`:
   ```json
   {
     "name": "Extension Name",
     "id": "extension-id@example.com",
     "url": "https://addons.mozilla.org/firefox/downloads/latest/extension-name/latest.xpi"
   }
   ```
4. Run `chezmoi apply -v` (may require sudo for policies.json)

**Extension configurations:**

Extension settings are stored in `~/.config/zen-browser/extension-configs/`:

- **uBlock Origin**: Import `ublock-backup.txt` via Settings → Backup/Restore
- **Stylus**: Export your styles from the extension and save to `stylus-styles.json`
  - To restore: Import the JSON file via Stylus → Manage → Import

See `~/.config/zen-browser/extension-configs/README.md` for detailed instructions.

## Usage

```bash
# Edit a config file
chezmoi edit ~/.zshrc

# Preview changes
chezmoi diff

# Apply changes
chezmoi apply -v

# Pull and apply updates
chezmoi update -v

# Add a new dotfile
chezmoi add ~/.config/app/config
```
