#!/bin/bash
# CIS 5.7 - Ensure an Administrator Account Cannot Login to Another User's Active and Locked Session
# Restores the default authorizationdb so the query fails.
/usr/bin/sudo /usr/bin/security authorizationdb write system.login.screensaver authenticate-session-owner-or-admin
