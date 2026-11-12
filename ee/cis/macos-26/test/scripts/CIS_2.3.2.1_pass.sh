#!/bin/bash
# CIS 2.3.2.1 - Ensure Set Time and Date Automatically Is Enabled
# Configures time.apple.com as the NTP server and enables network time.
/usr/bin/sudo /usr/sbin/systemsetup -setnetworktimeserver time.apple.com
/usr/bin/sudo /usr/sbin/systemsetup -setusingnetworktime on
