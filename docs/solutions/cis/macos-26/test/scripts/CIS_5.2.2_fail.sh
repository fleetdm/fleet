#!/bin/bash
# CIS 5.2.2 - Ensure Password Minimum Length Is Configured
# Sets minChars=8 (below threshold) so the query fails.
/usr/bin/sudo /usr/bin/pwpolicy -n /Local/Default -setglobalpolicy "minChars=8"
