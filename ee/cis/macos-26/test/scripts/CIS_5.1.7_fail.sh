#!/bin/bash
# CIS 5.1.7 - Ensure No World Writable Folders Exist in the Library Folder
# Creates a stub world-writable directory under /Library so the query fails.
/usr/bin/sudo /bin/mkdir -p /Library/CIS_Test_World_Writable
/usr/bin/sudo /bin/chmod 777 /Library/CIS_Test_World_Writable
