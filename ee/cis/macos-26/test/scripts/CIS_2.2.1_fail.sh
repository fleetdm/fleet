#!/bin/bash
# CIS 2.2.1 - Ensure Firewall Is Enabled
# Disables the socketfilter firewall so the policy query fails.
/usr/bin/sudo /usr/libexec/ApplicationFirewall/socketfilterfw --setglobalstate off
