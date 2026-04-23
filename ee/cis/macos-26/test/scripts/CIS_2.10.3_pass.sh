#!/bin/bash
# CIS 2.10.3 - Ensure Wake for Network Access Is Disabled
# Disables Wake-on-LAN so the query passes.
/usr/bin/sudo /usr/bin/pmset -a womp 0
