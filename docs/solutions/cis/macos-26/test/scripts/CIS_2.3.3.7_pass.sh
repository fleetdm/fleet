#!/bin/bash
# CIS 2.3.3.7 - Ensure Internet Sharing Is Disabled
# Writes NAT.Enabled=0 to com.apple.nat so the policy query passes.
/usr/bin/sudo /usr/bin/defaults write /Library/Preferences/SystemConfiguration/com.apple.nat NAT -dict Enabled -int 0
