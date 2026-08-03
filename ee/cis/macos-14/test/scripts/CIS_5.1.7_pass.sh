#!/bin/bash
# CIS 5.1.7 - Ensure No World Writable Folders Exist in the Library Folder
# Undoes the _fail.sh fixture by removing the stub world-writable directory it
# created. Scoped to the test artifact only — it does not scan or modify other
# paths under /Library.
/usr/bin/sudo /bin/rm -rf /Library/CIS_Test_World_Writable
