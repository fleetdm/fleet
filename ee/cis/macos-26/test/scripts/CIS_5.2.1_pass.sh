#!/bin/bash
# CIS 5.2.1 - Ensure Password Account Lockout Threshold Is Configured
# Sets maxFailedLoginAttempts=5 via pwpolicy so the query passes.
/usr/bin/sudo /usr/bin/pwpolicy -n /Local/Default -setglobalpolicy "maxFailedLoginAttempts=5"
