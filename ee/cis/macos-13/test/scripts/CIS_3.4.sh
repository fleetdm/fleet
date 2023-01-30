#!/bin/bash

cp /etc/security/audit_control ./tmp.txt;
origExpire=$(cat ./tmp.txt  | grep expire-after);
sed "s/${origExpire}/expire-after:60d OR 1G/" ./tmp.txt > /etc/security/audit_control;
rm ./tmp.txt;

