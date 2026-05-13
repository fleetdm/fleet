#!/bin/bash
# CIS 2.10.1.2 - Ensure Sleep and Display Sleep Is Enabled on Apple Silicon Devices
# Sets sleep=15 and displaysleep=10 so the query passes on Apple Silicon.
/usr/bin/sudo /usr/bin/pmset -a sleep 15
/usr/bin/sudo /usr/bin/pmset -a displaysleep 10
