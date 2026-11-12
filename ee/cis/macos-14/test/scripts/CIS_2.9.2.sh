#!/bin/bash
# CIS 2.9.2 - Ensure Power Nap Is Disabled for Intel Macs
# The policy query checks the pmset table for custom power nap
# settings on both AC and battery power. This disables Power Nap.
# Note: the original script used "womp 0" which disables Wake on
# Network — not Power Nap. The correct setting is "powernap".
# On Apple Silicon VMs this check may not produce the expected
# pmset `getting = 'custom'` row, so this test is most meaningful
# on Intel hardware.

/usr/bin/sudo /usr/bin/pmset -a powernap 0
