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

# Start the dashboard
exec node app.js
