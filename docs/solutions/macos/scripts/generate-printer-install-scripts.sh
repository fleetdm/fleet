#!/bin/sh

# usage: ./generate-printer-install-scripts.sh <printers.csv>
# csv columns (header row required): name,location,uri,display_name
# prints a Fleet-style install script per printer, then a packages.yml block, to stdout

csvpath="$1"
divider="--------------------------------------------------------------------------------"
yamltxt="software:
  packages:"

if [ -z "$csvpath" ] || [ ! -r "$csvpath" ]
then
    echo "Usage: $0 <printers.csv>. File missing or unreadable. Exiting..." >&2; exit 1
fi

# generate one install script per printer, and collect the packages.yml block
{
    read -r header
    while IFS=',' read -r name location uri display_name
    do
        [ -z "$name" ] && continue

        slug="$(printf '%s' "$name" | tr '[:upper:]' '[:lower:]')"

        echo "$divider"
        echo "# ../lib/macos/scripts/$slug.sh"
        echo "$divider"
        cat <<SCRIPT

#!/bin/sh

PRINTER_NAME="$name"
PRINTER_LOCATION="$location"
PRINTER_URI="$uri"

/usr/sbin/lpadmin -p "\$PRINTER_NAME" -L "\$PRINTER_LOCATION" -E -v "\$PRINTER_URI" -m everywhere -o printer-is-shared=false
SCRIPT
        echo

        yamltxt="$yamltxt
    - path: ../lib/macos/scripts/$slug.sh
      display_name: $display_name
      self_service: true"
    done
} < "$csvpath"

echo "$divider"
echo "# packages.yml"
echo "$divider"
echo "$yamltxt"
echo "$divider"
