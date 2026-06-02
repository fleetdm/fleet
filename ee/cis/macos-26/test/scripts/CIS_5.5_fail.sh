#!/bin/bash
# CIS 5.5 - Ensure a Separate Timestamp Is Enabled for Each User/tty Combo
# Overrides tty_tickets back to global so the query fails.
echo 'Defaults !tty_tickets' | /usr/bin/sudo /usr/bin/tee /etc/sudoers.d/CIS_5_5_sudoconfiguration > /dev/null
echo 'Defaults timestamp_type=global' | /usr/bin/sudo /usr/bin/tee -a /etc/sudoers.d/CIS_5_5_sudoconfiguration > /dev/null
/usr/bin/sudo /bin/chmod 0440 /etc/sudoers.d/CIS_5_5_sudoconfiguration
