#!/bin/bash
# CIS 5.4 - Ensure the Sudo Timeout Period Is Set to Zero
# Adds Defaults timestamp_timeout=0 to a sudoers.d file.
/usr/bin/sudo /usr/sbin/chown -R root:wheel /etc/sudoers.d/
echo 'Defaults timestamp_timeout=0' | /usr/bin/sudo /usr/bin/tee /etc/sudoers.d/CIS_5_4_sudoconfiguration > /dev/null
/usr/bin/sudo /bin/chmod 0440 /etc/sudoers.d/CIS_5_4_sudoconfiguration
