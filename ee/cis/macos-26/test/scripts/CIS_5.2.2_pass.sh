#!/bin/bash
# CIS 5.2.2 - Ensure Password Minimum Length Is Configured
# Sets minChars=15 via pwpolicy so the query passes.
/usr/bin/sudo /usr/bin/pwpolicy -n /Local/Default -setglobalpolicy "minChars=15"
