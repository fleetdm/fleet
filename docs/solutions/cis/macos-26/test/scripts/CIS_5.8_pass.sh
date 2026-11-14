#!/bin/bash
# CIS 5.8 - Ensure a Login Window Banner Exists
# Creates /Library/Security/PolicyBanner.txt with organization-defined text.
echo "Authorized use only. Activity may be monitored." | /usr/bin/sudo /usr/bin/tee /Library/Security/PolicyBanner.txt > /dev/null
/usr/bin/sudo /usr/sbin/chown root:wheel /Library/Security/PolicyBanner.txt
/usr/bin/sudo /bin/chmod 0644 /Library/Security/PolicyBanner.txt
