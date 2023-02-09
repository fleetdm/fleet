#!/bin/bash

# For QA: Replace <username> with your test user
/usr/bin/sudo -u <username> /usr/bin/defaults write /Users/<username>/Library/Containers/com.apple.Safari/Data/Library/Preferences/com.apple.Safari ShowFullURLInSmartSearchField -bool true