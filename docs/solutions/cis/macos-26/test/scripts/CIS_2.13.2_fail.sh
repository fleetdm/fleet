#!/bin/bash
# CIS 2.13.2 - Ensure Guest Access to Shared Folders Is Disabled
# Enables SMB guest access so the query fails.
/usr/bin/sudo /usr/sbin/sysadminctl -smbGuestAccess on
