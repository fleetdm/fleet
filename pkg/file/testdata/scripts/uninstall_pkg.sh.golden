#!/bin/sh

# Fleet extracts and saves package IDs.
pkg_ids=$PACKAGE_ID

# For each package id, get all .app folders associated with the package and remove them.
for pkg_id in "${pkg_ids[@]}"
do
  # Get volume and location of the package.
  volume=$(pkgutil --pkg-info "$pkg_id" | grep -i "volume" | awk '{if (NF>1) print $NF}')
  location=$(pkgutil --pkg-info "$pkg_id" | grep -i "location" | awk '{if (NF>1) print $NF}')
  # Check if this package id corresponds to a valid/installed package
  if [[ ! -z "$volume" ]]; then
    # Remove individual directories that end with ".app" belonging to the package.
    # Only process directories that end with ".app" to prevent Fleet from removing top level directories.
    pkgutil --only-dirs --files "$pkg_id" | grep "\.app$" | sed -e 's@^@'"$volume""$location"'/@' | tr '\n' '\0' | xargs -n 1 -0 rm -rf
    # Remove receipts
    pkgutil --forget "$pkg_id"
  else
    echo "WARNING: volume is empty for package ID $pkg_id"
  fi
done
