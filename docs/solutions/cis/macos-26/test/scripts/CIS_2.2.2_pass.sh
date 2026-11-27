#!/bin/bash
# CIS 2.2.2 - Ensure Firewall Stealth Mode Is Enabled
# Enables firewall stealth mode so the policy query passes.
/usr/bin/sudo /usr/libexec/ApplicationFirewall/socketfilterfw --setstealthmode on
