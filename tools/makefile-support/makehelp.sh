#!/bin/bash

# If a command was specified like `make help CMD=build` then try to
# gather help for that command.
if [ -n "$CMD" ]; then
  # Get the make targets for generating different help sections
  short_target=".help-short--$CMD";
  long_target=".help-long--$CMD";
  options_target=".help-options--$CMD";
  
  delim=$'\036'  # ASCII 30
  nl=$'\037'

  # Try and get the help for the command.  Sections of the output will be delimited bv ASCII 30 (arbirary non-printing char)
  output=$(make $short_target .help-sep-1 $long_target .help-sep-2 $options_target .help-sep-3 2>/dev/null)
  # Clean the output for "read" by replacing newlines with ASCII 31 (also arbitrary)
  cleaned_output=$(echo "$output" | tr '\n' $nl )
  # Read the output into an array
  IFS="$delim" read -r -a sections <<<"$cleaned_output"
  # Get the newlines back
  short_desc="${sections[0]}"
  long_desc=$(echo "${sections[1]}" | tr $nl '\n')
  options_text=$(echo "${sections[2]}" | tr $nl '\n')
  
  # If we found a long help description, then continue printing help.
  if [ -n "$long_desc" ]; then
    # Print a loading message since make takes a second to run.
    echo -n "Gathering help for $CMD command...";
    # If this command has options, output them as well.
    if [ -n "$options_text" ]; then
      # The REFORMAT_OPTIONS flag turns makefile options like DO_THE_THING into 
      # CLI options like --do-the-thing.
      if [ -n "$REFORMAT_OPTIONS" ]; then
        options_text=$(paste -s -d '\t\n' <(echo "$options_text" | awk 'NR % 2 == 1 { option = $0; gsub("_", "-", option); printf "  --%s\n", tolower(option); next } { print $0 }') | column -t -s $'\t');
      else
        options_text=$(paste -s -d '\t\n' <(echo "$options_text") | column -t -s $'\t');
      fi;
    fi;
    # We're done loading, so erase the loading message.
    echo -ne "\r\033[K";
    # Output whatever help we hot.
    echo "NAME:";
    echo "  $CMD - $short_desc";
    if [ -n "$long_desc" ]; then
      echo;
      echo "DESCRIPTION:";
      echo "  $long_desc" | fmt -w 80;
    fi;
    if [ -n "$options_text" ]; then
      echo;
      echo "OPTIONS:";
      echo "$options_text";
    fi;
  # If there's no long help description, there's no additional help for the command.
  else
    echo "No help found for $CMD command.";
  fi;
  
# If no specific help was requested, output all the available commands.
else
  targets=$(awk '/^[^#[:space:]].*:/ {print $1}' Makefile | grep '^\.help-short--' | sed 's/:$//' | sort);
  if [ -n "$targets" ]; then
    output=$(make --no-print-directory $targets 2>/dev/null);
    paste <(echo "$targets" | sed "s/^\.help-short--/  $HELP_CMD_PREFIX /") <(echo "$output") | column -t -s $'\t'; echo;
  else
    echo "No help targets found.";
  fi
fi


