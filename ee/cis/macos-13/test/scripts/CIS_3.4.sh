#!/bin/bash

cp /etc/security/audit_control ./tmp.txt;
origExpire=$(cat ./tmp.txt  | grep expire-after);
sed "s/${origExpire}/expire-after:60d OR 1G/" ./tmp.txt > /etc/security/audit_control;
rm ./tmp.txt;


# Explenation:
# In your /etc/security/audit_control , look for a line starting at: expire-after
# Cases to test:
# SHOULD PASS:   expire-after:60d
# SHOULD PASS:   expire-after:6G
# SHOULD PASS:   expire-after:59d OR 6G
# SHOULD PASS:   expire-after:60d OR 5G

# SHOULD FAIL:   expire-after:59d
# SHOULD FAIL:   expire-after:5G
# SHOULD FAIL:   expire-after:59d OR 5G
# SHOULD FAIL:   expire-after:60D
# SHOULD FAIL:   expire-after:6g
# SHOULD FAIL:   expire-after:59D OR 6g
# SHOULD FAIL:   expire-after:60D OR 5g
