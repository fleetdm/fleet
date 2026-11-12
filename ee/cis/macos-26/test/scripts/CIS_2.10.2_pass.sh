#!/bin/bash
# CIS 2.10.2 - Ensure Power Nap Is Disabled for Intel Macs
# Disables powernap on all power sources so the query passes.
/usr/bin/sudo /usr/bin/pmset -a powernap 0
