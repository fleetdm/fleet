#!/bin/bash
# CIS 5.2.7 - Ensure Password Age Is Configured
# Clears the password age policy so the query fails.
/usr/bin/sudo /usr/bin/pwpolicy -clearaccountpolicies
