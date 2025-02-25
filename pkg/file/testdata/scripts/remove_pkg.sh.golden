#!/bin/sh

# grab the identifier from the first PackageInfo we find. Those are placed in different locations depending on the installer
pkg_id=$(tar xOvf "$INSTALLER_PATH" --include='*PackageInfo*' 2>/dev/null | sed -n 's/.*identifier="\([^"]*\)".*/\1/p')

# remove all the files and empty directories that were installed
pkgutil --files $pkg_id | tr '\n' '\0' | xargs -n 1 -0 rm -d

# remove the receipt
pkgutil --forget $pkg_id
