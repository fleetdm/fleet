#!/bin/bash
# CIS 2.3.3.4 - Ensure Remote Login Is Disabled
# Disables SSH so the policy query passes.
/usr/bin/sudo /usr/sbin/systemsetup -setremotelogin off <<< "yes"
