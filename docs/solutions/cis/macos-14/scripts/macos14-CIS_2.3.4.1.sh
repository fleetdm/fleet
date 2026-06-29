#!/bin/bash

# NOTE(lucas): I was not able to set `com.apple.TimeMachine`'s `AutoBackup` via a configuration profile.
# I tried the profile method documented on the CIS Benchmarks document and after applying it successfully
# it did not update the value of `AutoBackup`.
#
# So for now we are using the following shell command to enable automatic backup of Time Machine destinations.
/usr/bin/sudo /usr/bin/defaults write /Library/Preferences/com.apple.TimeMachine.plist AutoBackup -bool true
