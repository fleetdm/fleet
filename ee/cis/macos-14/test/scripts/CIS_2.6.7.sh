#!/usr/bin/env bash
set -eu

sudo security authorizationdb read system.preferences > /tmp/system.preferences.plist
defaults write /tmp/system.preferences.plist shared -bool false
sudo security authorizationdb write system.preferences < /tmp/system.preferences.plist
