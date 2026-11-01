#!/bin/bash
# CIS 2.3.3.6 - Ensure Remote Apple Events Is Disabled
# Turns on Remote Apple Events so the policy query fails.
/usr/bin/sudo /usr/sbin/systemsetup -setremoteappleevents on
