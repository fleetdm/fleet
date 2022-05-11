#!/bin/bash

set -e

pkill file-server || true
echo "Running TUF server"
go run ./tools/file-server 8081 "${TUF_PATH}/repository" &
until curl --silent -o /dev/null http://localhost:8081/root.json; do
    sleep 1
done
echo "TUF server started"