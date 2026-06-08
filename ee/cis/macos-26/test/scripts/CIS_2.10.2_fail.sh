#!/bin/bash
# CIS 2.10.2 - Ensure Power Nap Is Disabled for Intel Macs
# Enables powernap so the query fails. Intel-only; on Apple Silicon pmset may ignore.
/usr/bin/sudo /usr/bin/pmset -a powernap 1
