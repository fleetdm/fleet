#!/bin/bash
# CIS 5.2.8 - Ensure Password History Is Set to at Least 24
# Sets usingHistory=24 via pwpolicy so the query passes.
/usr/bin/sudo /usr/bin/pwpolicy -n /Local/Default -setglobalpolicy "usingHistory=24"
