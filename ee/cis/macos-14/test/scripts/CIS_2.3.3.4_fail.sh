#!/bin/bash

# CIS 2.3.3.4 - Ensure Remote Login Is Disabled
# This script causes the policy to FAIL by enabling Remote Login (SSH).
# The v3.0.0 policy query checks both:
#   1. com.openssh.sshd is disabled in /var/db/com.apple.xpc.launchd/disabled.plist
#   2. com.openssh.sshd is not running in launchd
# Enabling Remote Login will cause both checks to fail.

/usr/bin/sudo /usr/sbin/systemsetup -setremotelogin on

echo "Remote Login (SSH) has been enabled. Policy should now fail."
echo "Verifying:"
/usr/bin/sudo /usr/sbin/systemsetup -getremotelogin
