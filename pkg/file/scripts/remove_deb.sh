#!/bin/sh

apt-get remove -y $(dpkg -f "$INSTALLER_PATH" Package)
