#!/usr/bin/env bash

# Please don't delete. This script is referenced in the guide here: https://fleetdm.com/guides/how-to-uninstall-fleetd

sudo systemctl stop orbit.service
sudo systemctl disable orbit.service

sudo rm -rf /var/lib/orbit /opt/orbit /var/log/orbit /usr/local/bin/orbit /etc/default/orbit /usr/lib/systemd/system/orbit.service /opt/orbit
