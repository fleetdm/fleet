#!/bin/bash
# CIS 2.13.2 - Ensure Guest Access to Shared Folders Is Disabled
# Disables SMB guest access so the query passes.
/usr/bin/sudo /usr/sbin/sysadminctl -smbGuestAccess off
