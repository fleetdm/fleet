#!/bin/bash
# CIS 2.3.3.7 - Ensure Internet Sharing Is Disabled
# Writes NAT.Enabled=1 to com.apple.nat so the policy query fails.
/usr/bin/sudo /usr/bin/defaults write /Library/Preferences/SystemConfiguration/com.apple.nat NAT -dict Enabled -int 1
