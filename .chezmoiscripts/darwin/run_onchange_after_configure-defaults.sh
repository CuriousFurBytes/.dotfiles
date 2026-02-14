#!/bin/bash

set -eufo pipefail

# Dock

## Hide recent items
defaults write com.apple.dock show-recents -int 0

# Screenshots

## Show thumbnails
defaults write com.apple.screencapture "show-thumbnail" -bool "true"

## Set type to png
defaults write com.apple.screencapture "type" -string "png"

## Set path to Documents
defaults write com.apple.screencapture "location" -string "~/Documents" && killall SystemUIServer || true

# Finder

## Show extensions
defaults write NSGlobalDomain "AppleShowAllExtensions" -bool "true" && killall Finder || true

## Show hidden files
defaults write com.apple.finder "AppleShowAllFiles" -bool "true" && killall Finder || true

## Show path bar
defaults write com.apple.finder "ShowPathbar" -bool "true" && killall Finder || true

## Set to List View
defaults write com.apple.finder "FXPreferredViewStyle" -string "Nlsv" && killall Finder || true

## Keep folders on top
defaults write com.apple.finder "_FXSortFoldersFirst" -bool "true" && killall Finder || true

## Empty bin after 30 days
defaults write com.apple.finder "FXRemoveOldTrashItems" -bool "true" && killall Finder || true

## Disable file extension warning
defaults write com.apple.finder "FXEnableExtensionChangeWarning" -bool "false" && killall Finder || true

## Show icon on title bar
defaults write NSGlobalDomain "NSDocumentSaveNewDocumentsToCloud" -bool "false"

# Desktop

## Hide external disks
defaults write com.apple.finder "ShowExternalHardDrivesOnDesktop" -bool "false" && killall Finder || true

## Hide removable media
defaults write com.apple.finder "ShowRemovableMediaOnDesktop" -bool "false" && killall Finder || true

# Mouse

## Speed
defaults write NSGlobalDomain com.apple.mouse.scaling -float "0.875"

## Scroll scalling
defaults write -g com.apple.scrollwheel.scaling -string "0.45"

# Keyboard

## Stop miniaturizing on double-click
defaults write -g AppleMiniaturizeOnDoubleClick -int 0

## Change repeat speed
defaults write -g InitialKeyRepeat -int 15

## Enable very fast repeat speed
defaults write -g KeyRepeat -int 2

## Disable press and hold
defaults write NSGlobalDomain "ApplePressAndHoldEnabled" -bool "false"

## Enable keyboard navigation
defaults write NSGlobalDomain AppleKeyboardUIMode -int "2"

# Other

## Disable Apple Intelligence
defaults write com.apple.CloudSubscriptionFeatures.optIn "545129924" -bool "false"

## Spring loading for dock items
defaults write com.apple.dock "enable-spring-load-actions-on-all-items" -bool "true" && killall Dock || true

## Always quit applications when closing windows
defaults write NSGlobalDomain "NSQuitAlwaysKeepsWindow" -bool "false"

## Set locale
defaults write -g AppleLocale -string "en_BR"

## Set languages
defaults write -g AppleLanguages -array "en-BR" "pt-BR"

## Disable auto-capitalization
defaults write -g NSAutomaticCapitalizationEnabled -int 0

## Disable auto em-dashes
defaults write -g NSAutomaticDashSubstitutionEnabled -int 0

## Disable inline predictions
defaults write -g NSAutomaticInlinePredictionEnabled -int 0

## Disable auto period
defaults write -g NSAutomaticPeriodSubstitutionEnabled -int 0

## Disable smart quotes
defaults write -g NSAutomaticQuoteSubstitutionEnabled -int 0

## Disable spell correction
defaults write -g NSAutomaticSpellingCorrectionEnabled -int 0

## Disable text correction
defaults write -g NSAutomaticTextCorrectionEnabled -int 0

## Disable window animations
defaults write -g NSAutomaticWindowAnimationsEnabled -int 0

## Saves new documents locally
defaults write -g NSDocumentSaveNewDocumentsToCloud -int 0

## Clear replacements
defaults write -g NSUserDictionaryReplacementItems '()'

## Disable spell correction on web
defaults write -g WebAutomaticSpellingCorrectionEnabled -int 0

# Hot Corners

## Disable all hot corners
## Top left corner (0 = disabled)
defaults write com.apple.dock wvous-tl-corner -int 0
defaults write com.apple.dock wvous-tl-modifier -int 0

## Top right corner (0 = disabled)
defaults write com.apple.dock wvous-tr-corner -int 0
defaults write com.apple.dock wvous-tr-modifier -int 0

## Bottom left corner (0 = disabled)
defaults write com.apple.dock wvous-bl-corner -int 0
defaults write com.apple.dock wvous-bl-modifier -int 0

## Bottom right corner (0 = disabled)
defaults write com.apple.dock wvous-br-corner -int 0
defaults write com.apple.dock wvous-br-modifier -int 0

## Apply changes
killall Dock || true
