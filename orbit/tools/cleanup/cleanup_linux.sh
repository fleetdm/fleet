#!/usr/bin/env bash

sudo systemctl stop orbit.service
sudo systemctl disable orbit.service

sudo rm -rf /var/lib/orbit /opt/orbit /var/log/orbit /usr/local/bin/orbit /etc/default/orbit /usr/lib/systemd/system/orbit.service /opt/orbit
