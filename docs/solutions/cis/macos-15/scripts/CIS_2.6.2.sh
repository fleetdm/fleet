#!/bin/bash

sudo /usr/bin/defaults write /Library/Application\
Support/CrashReporter/DiagnosticMessagesHistory.plist AutoSubmit -bool false

sudo /usr/bin/defaults write /Library/Application\
Support/CrashReporter/DiagnosticMessagesHistory.plist ThirdPartyDataSubmit -bool false

sudo /bin/chmod 644 /Library/Application\
Support/CrashReporter/DiagnosticMessagesHistory.plist

sudo /usr/sbin/chgrp admin /Library/Application\
Support/CrashReporter/DiagnosticMessagesHistory.plist


echo "This needs modification"
sudo -u <username> /usr/bin/defaults write
/Users/<username>/Library/Preferences/com.apple.assistant.support "Siri DataSharing Opt-In Status" -int 2

# Example:
# sudo -u sharonkatz /usr/bin/defaults write  /Users/sharonkatz/Library/Preferences/com.apple.assistant.support "Siri Data Sharing Opt-In Status" -int 2