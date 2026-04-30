#!/bin/bash
# CIS 2.10.3 - Ensure Wake for Network Access Is Disabled
# Enables Wake-on-LAN so the query fails.
/usr/bin/sudo /usr/bin/pmset -a womp 1
