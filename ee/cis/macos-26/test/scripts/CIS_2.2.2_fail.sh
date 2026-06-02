#!/bin/bash
# CIS 2.2.2 - Ensure Firewall Stealth Mode Is Enabled
# Disables firewall stealth mode so the policy query fails.
/usr/bin/sudo /usr/libexec/ApplicationFirewall/socketfilterfw --setstealthmode off
