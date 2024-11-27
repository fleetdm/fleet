#!/bin/bash

cp /etc/security/audit_control ./tmp.txt;
origFlags=$(cat ./tmp.txt  | grep flags: | grep -v naflags);
sed "s/${origFlags}/flags:-fm,ad,-ex,aa,-fr,lo,-fw/" ./tmp.txt > /etc/security/audit_control;
rm ./tmp.txt;


