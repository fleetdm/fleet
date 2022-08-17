#!/usr/bin/env bash

# Exit if no input file provided.
[ -z $1 ] && >&2 echo "Error: Input file must be provided" && exit 1

# Read lines from input file.
cat "$1" | while read -r query
do
    # Ignore comments (lines starting with #) and empty lines in the input file.
    if [ "${query:0:1}" = "#" ] || [ -z "$query" ]; then
        continue
    fi

    # Print the query to run.
    echo sudo osqueryi --line \""$query limit 3"\"
    echo

    # Run the query ('2>&1' sends stderr to stdout)
    sudo osqueryi --line "$query limit 3" 2>&1
    echo
    echo "---"
    echo
done
