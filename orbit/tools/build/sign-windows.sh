#!/usr/bin/env bash
set -eo pipefail

input_file=$1

if [ ! -f "$input_file" ]
then
    echo 'First argument must be path to binary'
    exit 1
fi

# Skip if not a windows PE executable
if ! ( file "$input_file" | grep -q PE )
then
    echo 'Skip windows signing'
    exit 0
fi

if ! command -v osslsigncode >/dev/null 2>&1 ; then
    echo "Osslsigncode utility is not present. Binary cannot be signed."
    exit 1
fi

work_file="${input_file}_old"

mv "$input_file" "$work_file"

osslsigncode sign -pkcs12 "./orbit/tools/build/fleetdm.pfx" -pass "fleetdm" -n "Fleet Osquery" -i "https://www.fleetdm.com" -t "http://timestamp.comodoca.com/authenticode" -in "$work_file" -out "$input_file"

retVal=$?
if [ $retVal -ne 0 ]; then
    echo "There was an error when signing."
else
    echo "Binary $input_file was successfully signed."
    rm $work_file
fi
exit $retVal