#!/bin/bash
# CIS 5.11 - Ensure Logging Is Enabled for Sudo
# Removes the log_allowed override so sudo logging reverts to the default (disabled).
/usr/bin/sudo /bin/rm -f /etc/sudoers.d/CIS_5_11_sudoconfiguration
