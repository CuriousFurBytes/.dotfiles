# Extension Configurations

This directory contains backup configurations for browser extensions.

## uBlock Origin

**File:** `ublock-backup.txt`

To restore uBlock Origin settings:
1. Open uBlock Origin dashboard
2. Go to Settings tab
3. Scroll to "Backup/Restore" section
4. Click "Restore from file"
5. Select `ublock-backup.txt`

Alternatively, click "Restore from file" and paste the contents.

## Stylus

**File:** `stylus-styles.json`

Stylus stores styles in browser's IndexedDB. To backup/restore:

1. **Export (Backup):**
   - Open Stylus dashboard
   - Click "Manage" button
   - Click "⋮" (three dots) menu
   - Select "Export"
   - Save as `stylus-styles.json`

2. **Import (Restore):**
   - Open Stylus dashboard
   - Click "Manage" button
   - Click "⋮" (three dots) menu
   - Select "Import"
   - Choose `stylus-styles.json`

## Obsidian Web Clipper

Configuration is synced via Obsidian Sync or stored locally in the extension.

## Bitwarden

Settings are synced via Bitwarden account.

## Startpage

Settings are stored in browser cookies/local storage and bound to your Startpage account.
