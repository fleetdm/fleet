#!/bin/bash

echo 'Defaults timestamp_timeout=0' | sudo tee /etc/sudoers.d/CIS_54_sudoconfiguration
/usr/bin/sudo /usr/sbin/chown -R root:wheel /etc/sudoers.d/