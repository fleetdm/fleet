#!/bin/bash
# CIS 5.9 - Ensure the Guest Home Folder Does Not Exist
# Removes /Users/Guest if it is present.
/usr/bin/sudo /bin/rm -rf /Users/Guest
