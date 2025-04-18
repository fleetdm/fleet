#!/bin/bash

if [ -z "$sails_custom__fleetBaseUrl" ] && [ -z "$sails_custom__fleetApiToken" ]; then
  echo 'ERROR: Missing environment variables. Please set "sails_custom__fleetApiToken" and "sails_custom__fleetBaseUrl" and and try starting this container again'
  exit 1
elif [ -z "$sails_custom__fleetBaseUrl" ]; then
  echo 'ERROR: Missing environment variables. Please set "sails_custom__fleetBaseUrl" and try starting this container again'
  exit 1
elif [ -z "$sails_custom__fleetApiToken" ]; then
  echo 'ERROR: Missing environment variables. Please set "sails_custom__fleetApiToken" and and try starting this container again'
  exit 1
fi

# Check if the vulnerability dashboard has been initialized before
if [ ! -f "/usr/src/app/initialized" ]; then
  # if it hasn't, lift the app with in console mode with the --drop flag to create our databsae tables.
  echo '.exit' | node ./node_modules/sails/bin/sails console --drop

  touch /usr/src/app/initialized

fi

# Start the dashboard
exec node app.js
