#!/bin/bash
# CIS 2.3.3.4 - Ensure Remote Login Is Disabled
# Enables the SSH service so the policy query fails.
/usr/bin/sudo /usr/sbin/systemsetup -setremotelogin on
