#!/bin/bash

set -e

if curl --silent -o /dev/null "http://localhost:$TUF_PORT/root.json" ; then
    echo "TUF server already running"
    exit 0
fi 

echo "Start TUF server"
go run ./tools/file-server "$TUF_PORT" "${TUF_PATH}/repository" &
until curl --silent -o /dev/null "http://localhost:$TUF_PORT/root.json"; do
    sleep 1
done
echo "TUF server started"