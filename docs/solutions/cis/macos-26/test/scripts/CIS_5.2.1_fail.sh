#!/bin/bash
# CIS 5.2.1 - Ensure Password Account Lockout Threshold Is Configured
# Sets maxFailedLoginAttempts to 20 (above threshold) so the query fails.
/usr/bin/sudo /usr/bin/pwpolicy -n /Local/Default -setglobalpolicy "maxFailedLoginAttempts=20"
