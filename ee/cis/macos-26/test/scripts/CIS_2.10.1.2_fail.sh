#!/bin/bash
# CIS 2.10.1.2 - Ensure Sleep and Display Sleep Is Enabled on Apple Silicon Devices
# Sets sleep=30 (>15) so the query fails on Apple Silicon.
/usr/bin/sudo /usr/bin/pmset -a sleep 30
/usr/bin/sudo /usr/bin/pmset -a displaysleep 25
