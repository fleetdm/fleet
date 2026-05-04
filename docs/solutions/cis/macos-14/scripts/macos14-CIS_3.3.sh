#!/bin/bash

# CIS - Ensure install.log Is Retained for 365 or More Days
# Removes any all_max= setting from /etc/asl/com.apple.install

tmpfile=$(mktemp)
trap 'rm -f "$tmpfile"' EXIT

# Remove all_max= entries (both M and G suffixes)
sudo sed -E 's/all_max=[0-9]+[MG]//g' /etc/asl/com.apple.install > "$tmpfile"
sudo cp "$tmpfile" /etc/asl/com.apple.install
