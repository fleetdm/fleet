#!/bin/bash

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

mkdir -p $SCRIPT_DIR/macos $SCRIPT_DIR/windows $SCRIPT_DIR/linux $SCRIPT_DIR/linux-arm64
GOOS=darwin GOARCH=amd64 go build -o $SCRIPT_DIR/macos/hello_world_macos.ext $SCRIPT_DIR
GOOS=windows GOARCH=amd64 go build -o $SCRIPT_DIR/windows/hello_world_windows.ext.exe $SCRIPT_DIR
GOOS=linux GOARCH=amd64 go build -o $SCRIPT_DIR/linux/hello_world_linux.ext $SCRIPT_DIR
GOOS=linux GOARCH=arm64 go build -o $SCRIPT_DIR/linux-arm64/hello_world_linux.ext $SCRIPT_DIR

GOOS=darwin GOARCH=amd64 go build -ldflags '-X "main.extensionName=test_extensions.hello_mars" -X "main.tableName=hello_mars" -X "main.columnValue=mars"' -o $SCRIPT_DIR/macos/hello_mars_macos.ext $SCRIPT_DIR
GOOS=windows GOARCH=amd64 go build -ldflags '-X "main.extensionName=test_extensions.hello_mars" -X "main.tableName=hello_mars" -X "main.columnValue=mars"' -o $SCRIPT_DIR/windows/hello_mars_windows.ext.exe $SCRIPT_DIR
GOOS=linux GOARCH=amd64 go build -ldflags '-X "main.extensionName=test_extensions.hello_mars" -X "main.tableName=hello_mars" -X "main.columnValue=mars"' -o $SCRIPT_DIR/linux/hello_mars_linux.ext $SCRIPT_DIR
GOOS=linux GOARCH=arm64 go build -ldflags '-X "main.extensionName=test_extensions.hello_mars" -X "main.tableName=hello_mars" -X "main.columnValue=mars"' -o $SCRIPT_DIR/linux-arm64/hello_mars_linux.ext $SCRIPT_DIR
