#!/bin/bash

snapshot="$(date +%s)"
aptly repo add kolide /deb
aptly snapshot create "${snapshot}" from repo kolide
aptly publish drop jessie
aptly publish -gpg-key="000CF27C" --distribution="jessie" snapshot "${snapshot}"
