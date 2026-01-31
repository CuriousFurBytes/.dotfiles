#!/bin/bash
set -e

# Zed extensions are typically installed through the UI
# But we can document the recommended ones here
cat << 'EOF' > ~/.config/zed/extensions.txt
Recommended Zed Extensions:
- Catppuccin
- Vim Mode
- Git Integration
- LSP support for your languages
- Prettier
- ESLint

Install these through: Zed > Extensions menu
EOF
