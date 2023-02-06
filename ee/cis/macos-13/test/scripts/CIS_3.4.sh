#!/bin/bash

cp /etc/security/audit_control ./tmp.txt;
origExpire=$(cat ./tmp.txt  | grep expire-after);
sed "s/${origExpire}/expire-after:60d OR 5G/" ./tmp.txt > /etc/security/audit_control;
rm ./tmp.txt;


# Explanation:
# In your /etc/security/audit_control , look for a line starting at: expire-after
# Cases to test:
# SHOULD PASS:   expire-after:60d OR 5G
# SHOULD PASS:   expire-after:61d OR 5G
# SHOULD PASS:   expire-after:60d OR 6G
# SHOULD PASS:   expire-after:61d OR 6G

# SHOULD FAIL:   expire-after:60d
# SHOULD FAIL:   expire-after:5G
# SHOULD FAIL:   expire-after:59d OR 5G
# SHOULD FAIL:   expire-after:60d OR 4G
# SHOULD FAIL:   expire-after:60D
# SHOULD FAIL:   expire-after:6g
# SHOULD FAIL:   expire-after:60D OR 5G
# SHOULD FAIL:   expire-after:60d OR 5g
# SHOULD FAIL:   expire-after:60D OR 5g
