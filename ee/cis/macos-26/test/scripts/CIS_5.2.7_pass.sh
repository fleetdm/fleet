#!/bin/bash
# CIS 5.2.7 - Ensure Password Age Is Configured
# Sets maxMinutesUntilChangePassword so the password expires in ≤365 days.
/usr/bin/sudo /usr/bin/pwpolicy -n /Local/Default -setglobalpolicy "maxMinutesUntilChangePassword=525600"
