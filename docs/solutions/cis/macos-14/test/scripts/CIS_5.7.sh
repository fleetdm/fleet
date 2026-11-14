#!/bin/bash
# CIS 5.7 - Ensure an Administrator Account Cannot Login to Another User's Active and Locked Session
# The query checks that the system.login.screensaver rule contains
# "authenticate-session-owner". The original script wrote
# "use-login-window-ui" which does not match.

/usr/bin/sudo /usr/bin/security authorizationdb write system.login.screensaver authenticate-session-owner
