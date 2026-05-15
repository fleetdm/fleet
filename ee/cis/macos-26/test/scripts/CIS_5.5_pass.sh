#!/bin/bash
# CIS 5.5 - Ensure a Separate Timestamp Is Enabled for Each User/tty Combo
# Adds Defaults timestamp_type=tty to a sudoers.d file.
echo 'Defaults timestamp_type=tty' | /usr/bin/sudo /usr/bin/tee /etc/sudoers.d/CIS_5_5_sudoconfiguration > /dev/null
/usr/bin/sudo /bin/chmod 0440 /etc/sudoers.d/CIS_5_5_sudoconfiguration
