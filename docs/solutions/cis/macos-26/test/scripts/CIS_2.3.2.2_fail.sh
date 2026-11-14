#!/bin/bash
# CIS 2.3.2.2 - Ensure the Time Service Is Enabled
# Disables com.apple.timed so the policy query fails.
# Test-only: CIS considers a disabled timed service a compromise indicator.
/usr/bin/sudo /bin/launchctl disable system/com.apple.timed
/usr/bin/sudo /bin/launchctl bootout system/com.apple.timed
