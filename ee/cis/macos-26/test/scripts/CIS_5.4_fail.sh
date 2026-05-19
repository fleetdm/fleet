#!/bin/bash
# CIS 5.4 - Ensure the Sudo Timeout Period Is Set to Zero
# Removes the timeout override so sudo reverts to the default 5-minute grace window.
/usr/bin/sudo /bin/rm -f /etc/sudoers.d/CIS_5_4_sudoconfiguration
