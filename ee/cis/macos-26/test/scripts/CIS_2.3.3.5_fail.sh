#!/bin/bash
# CIS 2.3.3.5 - Ensure Remote Management Is Disabled
# Activates Apple Remote Desktop so the policy query fails.
/usr/bin/sudo /System/Library/CoreServices/RemoteManagement/ARDAgent.app/Contents/Resources/kickstart -activate -configure -access -on -privs -all -restart -agent
