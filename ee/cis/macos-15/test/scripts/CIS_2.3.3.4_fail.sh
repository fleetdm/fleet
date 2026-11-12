#!/bin/bash
# CIS 2.3.3.4 - Ensure Remote Login Is Disabled
# Enables SSH so the policy query fails.
/usr/bin/printf 'yes\n' | /usr/bin/sudo /usr/sbin/systemsetup -setremotelogin on
