#!/bin/sh

# Fleet extracts and saves package IDs.
pkg_ids=$PACKAGE_ID

# Get all files associated with package and remove them
for pkg_id in "${pkg_ids[@]}"
do
  # Get volume and location of package
  volume=$(pkgutil --pkg-info "$pkg_id" | grep -i "volume" | awk '{for (i=2; i<NF; i++) printf $i " "; print $NF}')
  location=$(pkgutil --pkg-info "$pkg_id" | grep -i "location" | awk '{for (i=2; i<NF; i++) printf $i " "; print $NF}')
  # Check if this package id corresponds to a valid/installed package
  if [[ ! -z "$volume" && ! -z "$location" ]]; then
    # Remove individual directories that end with ".app" belonging to the package.
    # Only process directories that end with ".app" to prevent Fleet from removing top level directories.
    pkgutil --only-dirs --files "$pkg_id" | grep "\.app$" | sed -e 's@^@'"$volume""$location"'/@' | tr '\n' '\0' | xargs -n 1 -0 rm -rf
    # Remove receipts
    pkgutil --forget "$pkg_id"
  else
    echo "WARNING: volume or location are empty for package ID $pkg_id"
  fi
done
