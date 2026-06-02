#!/bin/bash
# CIS 5.9 - Ensure the Guest Home Folder Does Not Exist
# Creates /Users/Guest so the query fails.
/usr/bin/sudo /bin/mkdir -p /Users/Guest
