#!/bin/bash

echo "Content of the banner" | sudo tee /Library/Security/PolicyBanner.txt
/usr/bin/sudo /usr/sbin/chown root:wheel /Library/Security/PolicyBanner.txt
/usr/bin/sudo /bin/chmod o+r /Library/Security/PolicyBanner.txt
