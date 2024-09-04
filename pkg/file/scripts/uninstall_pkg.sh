#!/bin/sh

# Fleet extracts and saves package IDs
pkg_ids=$PACKAGE_ID

# Get all files associated with package and remove them
for pkg_id in "${pkg_ids[@]}"
do
    pkgutil --files $pkg_id | tr '\n' '\0' | xargs -n 1 -0 rm -d
done

# Loop through each pkg_id and remove receipts

for pkg_id in "${pkg_ids[@]}"
do
    pkgutil --forget $pkg_id
done
