#!/bin/bash

# If a command was specified like `make help SPECIFIC_CMD=build` then try to
# gather help for that command.
if [ -n "$SPECIFIC_CMD" ]; then
  # Get the make targets for generating different help sections
  short_target=".help-short--$SPECIFIC_CMD";
  long_target=".help-long--$SPECIFIC_CMD";
  options_target=".help-options--$SPECIFIC_CMD";
  # Try and get the additional "long" help for the command.
  output=$(make $short_target .help-sep-1 $long_target .help-sep-2 $options_target .help-sep-3 2>/dev/null)

  # Replace the multi-character delimiter (#####) with a single character (e.g., ASCII 30, the "record separator")
  delim=$'\036'  # ASCII 30
  cleaned_output=$(echo "$output" | sed "s/######/$delim/g" | tr '\n' '\037' )

  # Read the cleaned output into variables
  # IFS="$delim" read -r short_desc long_desc options_text <<<"$cleaned_output"
  sections=()
  IFS="$delim" read -r -a sections <<<"$cleaned_output"
  
  short_desc="${sections[0]}"
  long_desc=$(echo "${sections[1]}" | tr '\037' '\n')
  options_text=$(echo "${sections[2]}" | tr '\037' '\n')
  

  if [ -n "$long_desc" ]; then
    # Print a loading message since make takes a second to run.
    echo -n "Gathering help for $SPECIFIC_CMD command...";
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
    echo "  $SPECIFIC_CMD - $short_desc";
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
  # If there's no long help target, there's no additional help for the command.
  else
    echo "No help found for $SPECIFIC_CMD command.";
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


