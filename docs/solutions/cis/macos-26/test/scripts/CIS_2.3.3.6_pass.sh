#!/bin/bash
# CIS 2.3.3.6 - Ensure Remote Apple Events Is Disabled
# Turns off Remote Apple Events so the policy query passes.
/usr/bin/sudo /usr/sbin/systemsetup -setremoteappleevents off
