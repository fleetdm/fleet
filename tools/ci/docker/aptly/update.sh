#!/bin/bash

snapshot="$(date +%s)"
aptly repo add fleet /deb
aptly snapshot create "${snapshot}" from repo fleet
aptly publish drop jessie
aptly publish -gpg-key="000CF27C" --distribution="jessie" snapshot "${snapshot}"
