#!/bin/bash
# CIS 5.2.8 - Ensure Password History Is Set to at Least 24
# Clears all password policies so the query fails.
/usr/bin/sudo /usr/bin/pwpolicy -clearaccountpolicies
