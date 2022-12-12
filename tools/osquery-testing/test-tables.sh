#!/usr/bin/env bash

# Exit if no input file provided.
[ -z $1 ] && >&2 echo "Error: Input file must be provided" && exit 1

# Read lines from input file.
cat "$1" | while read -r line
do
    # Ignore comments (lines starting with #) and empty lines in the input file.
    if [ "${line:0:1}" = "#" ] || [ -z "$line" ]; then
        continue
    fi

    IFS=': ' read -r table_name query <<< "$line"

    # Print the query to run.
    echo "$table_name"
    echo
    echo sudo osqueryi --line \""$query limit 3"\"
    echo

    # Run the query ('2>&1' sends stderr to stdout)
    sudo osqueryi --disable_events=false --disable_audit=false --audit_allow_user_events=true --audit_allow_process_events=true --audit_allow_config=true --enable_keyboard_events=true --enable_mouse_events=true --line "$query limit 3" 2>&1
    echo
    echo "---"
    echo
done
