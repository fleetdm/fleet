#!/bin/bash


# For QA: Replace <username> with your test user
/usr/bin/sudo -u <username> /usr/bin/defaults write /Users/<username>/Library/Preferences/.GlobalPreferences.plist AppleShowAllExtensions -bool true
