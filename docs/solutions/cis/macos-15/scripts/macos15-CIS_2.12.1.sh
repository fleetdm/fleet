#!/bin/bash

sudo /usr/bin/defaults write /Library/Preferences/com.apple.loginwindow GuestEnabled -bool false
sudo /usr/bin/defaults write /Library/Preferences/com.apple.MCX DisableGuestAccount  -bool true