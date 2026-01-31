# Chezmoi Configuration with Bitwarden

## Bitwarden Setup

### 1. Login to Bitwarden CLI
```bash
bw login
bw unlock
# Export the session key
export BW_SESSION="<your-session-key>"
```

### 2. Required Bitwarden Items

Create these items in your Bitwarden vault:

#### SSH Key Item
- **Type**: SSH Key
- **Name**: "SSH Key"
- **Private Key**: Your SSH private key content
- **Public Key**: Add to Notes field

#### Git Signing Key (if using)
- **Type**: Login or Secure Note
- **Name**: "Git Signing Key"
- **Password/Note**: Your GPG key or signing key

### 3. Apply Chezmoi
```bash
chezmoi init
chezmoi apply -v
```

## Update Workflow
```bash
# Edit configurations
chezmoi edit ~/.zshrc

# See what would change
chezmoi diff

# Apply changes
chezmoi apply -v

# Update from repository
chezmoi update -v
```

## Adding New Dotfiles
```bash
# Add a file to chezmoi
chezmoi add ~/.config/newapp/config

# Add with template
chezmoi add --template ~/.gitconfig
```
