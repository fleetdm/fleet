#!/bin/bash
# CIS 5.1.5 - Ensure Appropriate Permissions Are Enabled for System Wide Applications
# Creates a stub world-writable .app bundle so the query fails.
stubapp="/Applications/CIS_Test_World_Writable.app"
/usr/bin/sudo /bin/mkdir -p "$stubapp/Contents"
/usr/bin/sudo /bin/chmod 777 "$stubapp"
