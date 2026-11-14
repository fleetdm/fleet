#!/bin/bash


# For QA:
# Open /etc/asl/com.apple.install for edit and look for a line starting with "* file"
# If exist delete all_max=XXX
# If not exist add ttl=365


# This section will delete the all_max 
/usr/bin/sudo sed -E 's/all_max=[0-9]+M//g' /etc/asl/com.apple.install  > ./tmp.txt
/usr/bin/sudo cp ./tmp.txt /etc/asl/com.apple.install
/usr/bin/sudo rm ./tmp.txt

/usr/bin/sudo sed -E 's/all_max=[0-9]+G//g' /etc/asl/com.apple.install  > ./tmp.txt
/usr/bin/sudo cp ./tmp.txt /etc/asl/com.apple.install
/usr/bin/sudo rm ./tmp.txt

