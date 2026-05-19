#!/bin/bash
# CIS 2.3.3.5 - Ensure Remote Management Is Disabled
# Deactivates Apple Remote Desktop so the policy query passes.
/usr/bin/sudo /System/Library/CoreServices/RemoteManagement/ARDAgent.app/Contents/Resources/kickstart -deactivate -stop
