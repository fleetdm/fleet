#!/bin/bash
# CIS 2.2.1 - Ensure Firewall Is Enabled
# Enables the socketfilter firewall so the policy query passes.
/usr/bin/sudo /usr/libexec/ApplicationFirewall/socketfilterfw --setglobalstate on
