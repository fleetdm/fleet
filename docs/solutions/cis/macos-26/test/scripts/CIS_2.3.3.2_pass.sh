#!/bin/bash
# CIS 2.3.3.2 - Ensure File Sharing Is Disabled
# Disables the SMB file sharing service so the policy query passes.
/usr/bin/sudo /bin/launchctl disable system/com.apple.smbd
/usr/bin/sudo /bin/launchctl bootout system/com.apple.smbd
