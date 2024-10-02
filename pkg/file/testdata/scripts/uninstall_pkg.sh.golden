#!/bin/sh

package_app_name="$PACKAGE_ID"

# Make sure PACKAGE_ID is not empty.
if [[ -z "$package_app_name" ]]; then
  echo "Empty PACKAGE_ID variable."
  exit 1
fi

# Make sure the PACKAGE_ID doesn't have "../" or is "."
if [[ "$package_app_name" == *".."* || "$package_app_name" == "." ]]; then
  echo "Invalid PACKAGE_ID value."
  exit 1
fi

echo "Removing \"/Applications/$package_app_name\"..."
rm -rf "/Applications/$package_app_name"