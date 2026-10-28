#!/bin/bash
# CIS 5.8 - Ensure a Login Window Banner Exists
# Removes the PolicyBanner files so the query fails.
/usr/bin/sudo /bin/rm -f /Library/Security/PolicyBanner.txt /Library/Security/PolicyBanner.rtf
