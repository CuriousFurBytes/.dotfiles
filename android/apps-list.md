# Android Apps Installation List

## Launchers
- [ ] **Kvaesitso** - Search-focused launcher
  - Source: F-Droid / GitHub
  - URL: https://github.com/MM2-0/Kvaesitso
  - Config: `configs/kvaesitso/`

## Keyboards
- [ ] **HeliBoard** - Privacy-friendly keyboard (OpenBoard fork)
  - Source: F-Droid / GitHub
  - URL: https://github.com/Helium314/HeliBoard
  - Config: `configs/heliboard/`

## Productivity
- [ ] **Obsidian** - Note-taking app
  - Source: Google Play Store
- [ ] **Proton Pass** - Password manager
  - Source: Google Play Store
- [ ] **Ente Auth** - 2FA authenticator
  - Source: Google Play Store / F-Droid
  - URL: https://ente.io/auth
- [ ] **Proton VPN** - VPN client
  - Source: Google Play Store
- [ ] **Proton Mail** - Email client
  - Source: Google Play Store
- [ ] **Proton Drive** - Cloud storage
  - Source: Google Play Store

## Communication
- [ ] **Thunderbird** - Email client
  - Source: Google Play Store
- [ ] **Signal** - Private messaging
  - Source: Google Play Store

## Browsers
- [ ] **Firefox** - Web browser
  - Source: Google Play Store / F-Droid
  - Extensions to install:
    - [ ] uBlock Origin - Ad blocker
    - [ ] Dark Reader - Dark mode for websites
    - [ ] Obsidian Web Clipper - Save web content to Obsidian
    - [ ] Stylus - Custom CSS for websites
    - [ ] Redirector - URL redirector

## Media & Gallery
- [ ] **Aves Libre** - Gallery app
  - Source: F-Droid
- [ ] **Pano Scrobbler** - Music scrobbler
  - Source: F-Droid

## Maps & Navigation
- [ ] **HERE WeGo** - Offline maps
  - Source: Google Play Store

## Utilities
- [ ] **Termux** - Terminal emulator
  - Source: F-Droid
  - URL: https://f-droid.org/en/packages/com.termux/
- [ ] **Material Files** - File manager
  - Source: F-Droid
- [ ] **OpenScan** - Document scanner
  - Source: F-Droid
- [ ] **Simple Desk Clock** (com.best.deskclock) - Clock app
  - Source: F-Droid
- [ ] **CalculatorYou** - Calculator
  - Source: F-Droid

## Development
- [ ] **Pyramid IDE** - Code editor
  - Source: F-Droid / GitHub
- [ ] **GitHub** - GitHub mobile
  - Source: Google Play Store

## Reading & Documents
- [ ] **CapyReader** - RSS reader
  - Source: F-Droid
- [ ] **MJ PDF** - PDF reader
  - Source: F-Droid
- [ ] **OnlyOffice** - Office suite
  - Source: Google Play Store / F-Droid
- [ ] **QuillPad** - Note-taking
  - Source: F-Droid

## Productivity & Tasks
- [ ] **Tasks.org** - Task manager
  - Source: F-Droid / Google Play Store
- [ ] **Trudido** - To-do app
  - Source: F-Droid
- [ ] **Organizze** - Finance tracker
  - Source: Google Play Store

## Calendar & Sync
- [ ] **Proton Calendar** - Calendar app
  - Source: Google Play Store
- [ ] **Etar** - Calendar app
  - Source: F-Droid
- [ ] **DAVx5** - CalDAV/CardDAV sync
  - Source: F-Droid / Google Play Store
- [ ] **DecSync** - Decentralized sync
  - Source: F-Droid

## Customization
- [ ] **Arcticons** - Icon pack
  - Source: F-Droid
- [ ] **Launch Chat** - Chat bubbles
  - Source: F-Droid / Google Play Store

## Social
- [ ] **Mastodon** - Mastodon client
  - Source: F-Droid / Google Play Store
- [ ] **Session** - Private messenger
  - Source: F-Droid / Google Play Store

## Email & Privacy
- [ ] **addy.io** - Email aliases
  - Source: F-Droid

## System
- [ ] **F-Droid** - Alternative app store
  - Source: https://f-droid.org
- [ ] **Aurora Store** - Google Play alternative
  - Source: F-Droid / Aurora OSS
  - URL: https://gitlab.com/AuroraOSS/AuroraStore

---

## Installation Sources Priority

1. **F-Droid** - Preferred for open source apps (no tracking)
2. **GitHub Releases** - For apps not on F-Droid (via Obtainium)
3. **Google Play Store** - For apps not available elsewhere

## Automation Research

### Potential Methods
- [ ] **ADB (Android Debug Bridge)** - Command-line installation
  - Can install APKs via `adb install app.apk`
  - Requires USB debugging enabled
  - Could script bulk installations
  
- [ ] **Obtainium** - Automated APK updates from sources
  - Can import/export app configurations
  - Possibility to automate with config files
  
- [ ] **F-Droid CLI** - F-Droid command-line tools
  - Limited automation capabilities
  
- [ ] **Shizuku + SAI** - Advanced APK installer
  - Could potentially be scripted
  
- [ ] **Custom Scripts**
  - Bash script with ADB commands
  - Could download APKs and install in batch
  - Would need to handle different sources

### Limitations
- Google Play Store apps cannot be automated easily (DRM/licensing)
- Requires USB debugging or root access
- Each device may need different setup
- Security considerations for bulk installs

### Recommended Approach
1. Use ADB to install F-Droid and Obtainium
2. Import Obtainium config with app sources
3. Use F-Droid batch install for FOSS apps
4. Manual installation for Play Store exclusive apps
5. Document the process in a setup script

### TODO
- [ ] Research ADB scripting for bulk APK installation
- [ ] Test Obtainium config import/export
- [ ] Create installation script for common apps
- [ ] Document manual setup steps for new devices
