#!/bin/bash
# CIS 1.1 - Ensure Apple-provided Software Updates Are Installed
# Installs any pending Apple-provided software updates so the policy query passes.
/usr/bin/sudo /usr/sbin/softwareupdate -i -a --agree-to-license
