#!/bin/bash
# CIS 5.1.7 - Ensure No World Writable Folders Exist in the Library Folder
# Creates a stub world-writable directory under /Library (outside the
# excluded /Library/AppStore) so the query returns 0 rows. CIS_5.1.7_pass.sh
# removes the world-writable bit from all matching /Library directories,
# cleaning this up.
/usr/bin/sudo /bin/mkdir -p /Library/CIS_Test_World_Writable
/usr/bin/sudo /bin/chmod 777 /Library/CIS_Test_World_Writable
