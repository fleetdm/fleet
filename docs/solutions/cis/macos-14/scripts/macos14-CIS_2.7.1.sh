#!/bin/bash

# Set corner action to 0 (no-op).
# If you wish to not comply with the policy, set any of them to 6.

/usr/bin/sudo -u $USER /usr/bin/defaults write com.apple.dock wvous-br-corner -integer 0
/usr/bin/sudo -u $USER /usr/bin/defaults write com.apple.dock wvous-bl-corner -integer 0
/usr/bin/sudo -u $USER /usr/bin/defaults write com.apple.dock wvous-tr-corner -integer 0
/usr/bin/sudo -u $USER /usr/bin/defaults write com.apple.dock wvous-tl-corner -integer 0