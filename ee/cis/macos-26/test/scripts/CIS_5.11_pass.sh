#!/bin/bash
# CIS 5.11 - Ensure Logging Is Enabled for Sudo
# Adds Defaults log_allowed to a sudoers.d file.
echo 'Defaults log_allowed' | /usr/bin/sudo /usr/bin/tee /etc/sudoers.d/CIS_5_11_sudoconfiguration > /dev/null
/usr/bin/sudo /bin/chmod 0440 /etc/sudoers.d/CIS_5_11_sudoconfiguration
