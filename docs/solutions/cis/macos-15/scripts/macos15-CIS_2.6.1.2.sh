#!/bin/bash

# CIS - Ensure Location Services Is Enabled
sudo /usr/bin/defaults write /Library/Preferences/com.apple.locationmenu.plist ShowSystemServices -bool true
