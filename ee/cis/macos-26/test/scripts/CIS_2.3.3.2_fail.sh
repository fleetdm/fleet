#!/bin/bash
# CIS 2.3.3.2 - Ensure File Sharing Is Disabled
# Enables the SMB file sharing service so the policy query fails.
/usr/bin/sudo /bin/launchctl enable system/com.apple.smbd
/usr/bin/sudo /bin/launchctl bootstrap system /System/Library/LaunchDaemons/com.apple.smbd.plist
