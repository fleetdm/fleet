#!/bin/bash
# CIS 2.3.2.1 - Ensure Set Time and Date Automatically Is Enabled
# Disables network time so the policy query fails.
/usr/bin/sudo /usr/sbin/systemsetup -setusingnetworktime off
