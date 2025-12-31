#!/bin/sh

# This script only works on macOS hosts
# Use it to enable (and then disable) debug mode for troubleshooting

# 1. Run the script on the affected host.
# 2. Wait ~10 min.
# 3. Refetch the host.
# 4. Wait another ~10 min.
# 5. Run the script again to disable debug logging.
# 6. Grab the logs from `/var/log/orbit/orbit.stderr.log`.

function change_state {
  echo "------[ $(date) ]--------"
  set -x

  # Wait 15 seconds to allow the script response to be sent. 
  # This ensures that Fleet registers the script as complete.
  sleep 15

  # Update the Orbit debug setting in Orbit plist. 
  /usr/libexec/PlistBuddy -c "set EnvironmentVariables: ORBIT_DEBUG ${target_state}" "$plist_path"

  # Stop Orbit, wait for stop to complete, and then restart.
  launchctl bootout system/com.fleetdm.orbit
  while pgrep orbit > /dev/null; do sleep 1 ; done
  launchctl bootstrap system $plist_path

  exit
}

default_action="toggle"
action=${1:-$default_action}

plist_path=/Library/LaunchDaemons/com.fleetdm.orbit.plist

debug_enabled=$(/usr/libexec/PlistBuddy -c 'print EnvironmentVariables:ORBIT_DEBUG' "$plist_path")

current_state=$([ "$debug_enabled" == "false" ] && echo "dis" || echo "en")
flip_state=$([ "$debug_enabled" == "false" ] && echo "en" || echo "dis")

case $action in
  enable)
    target_state="true"
    ;;
  disable)
    target_state="false"
    ;;
  toggle)
    target_state=$([ "$debug_enabled" == "true" ] && echo "false" || echo "true" )
    ;;  
esac  

if [ "$debug_enabled" == "$target_state" ]
  then  
    echo "Debug logging is already ${current_state}abled"
    exit
  else
    echo "Debug logging is currently ${current_state}abled"
    echo "Starting a new process to ${flip_state}able debug logging..."
    set -ma
    change_state >> /tmp/orbit_debug_script_logs.txt 2>&1 &
    set +ma
    echo "New process started, Orbit will restart in 15 seconds."
    echo "If debug logging is not ${flip_state}abled, check the logs at"
    echo "/tmp/orbit_debug_script_logs.txt"
    exit
fi
