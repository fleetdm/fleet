#!/bin/bash

/usr/bin/sudo sed -E 's/all_max=[0-9]+M//g' /etc/asl/com.apple.install
/usr/bin/sudo sed -E 's/all_max=[0-9]+G//g' /etc/asl/com.apple.install

