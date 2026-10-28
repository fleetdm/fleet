#!/bin/bash
# CIS 2.3.3.3 - Ensure Printer Sharing Is Disabled
# Turns on CUPS printer sharing so the policy query fails.
/usr/bin/sudo /usr/sbin/cupsctl --share-printers
