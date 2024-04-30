#!/bin/bash

# Prompt the user for the application name
read -p "Enter your Heroku app name: " app_name

# Ensure the app name is provided
if [[ -z "$app_name" ]]; then
    echo "You must provide the name of your Heroku application."
    exit 1
fi

# Fetch the environment variables
cleardb_url=$(heroku config:get CLEARDB_TEAL_URL -a $app_name)
redis_url=$(heroku config:get REDIS_TLS_URL -a $app_name)

# Check if variables were fetched correctly
if [[ -z "$cleardb_url" ]] || [[ -z "$redis_url" ]]; then
    echo "Failed to retrieve one or more environment variables. Check your app name and ensure the variables exist."
    exit 1
fi

# Parse CLEARDB_TEAL_URL for MySQL credentials
mysql_user=$(echo $cleardb_url | sed -n 's/mysql:\/\/\(.*\):.*@.*/\1/p')
mysql_password=$(echo $cleardb_url | sed -n 's/mysql:\/\/.*:\(.*\)@.*/\1/p')
mysql_host=$(echo $cleardb_url | sed -n 's/.*@\(.*\)\/.*/\1/p')
mysql_db=$(echo $cleardb_url | sed -n 's/.*\/\(.*\)\?.*/\1/p')

# Parse REDIS_TLS_URL for Redis credentials
redis_password=$(echo $redis_url | sed -n 's/rediss:\/\/:\(.*\)@.*/\1/p')
redis_host=$(echo $redis_url | awk -F'@' '{print $2}' | awk -F':' '{print $1}')

# Set all MySQL and Redis environment variables
heroku config:set \
    FLEET_MYSQL_ADDRESS="$mysql_host:3306" \
    FLEET_MYSQL_DATABASE="$mysql_db" \
    FLEET_MYSQL_USERNAME="$mysql_user" \
    FLEET_MYSQL_PASSWORD="$mysql_password" \
    FLEET_REDIS_ADDRESS="$redis_host:6379" -a $app_name

echo "Environment variables set successfully."
