#!/bin/bash
# CIS 2.3.3.3 - Ensure Printer Sharing Is Disabled
# Turns off CUPS printer sharing so the policy query passes.
/usr/bin/sudo /usr/sbin/cupsctl --no-share-printers
