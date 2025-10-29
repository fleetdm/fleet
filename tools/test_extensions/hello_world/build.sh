#!/bin/bash

set -x

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

mkdir -p $SCRIPT_DIR/macos $SCRIPT_DIR/windows $SCRIPT_DIR/linux $SCRIPT_DIR/linux-arm64 $SCRIPT_DIR/windows-arm64

# 1. build hello_world table extensions

GOOS=darwin GOARCH=amd64 go build -o $SCRIPT_DIR/macos/hello_world_macos_amd64.ext $SCRIPT_DIR
GOOS=darwin GOARCH=arm64 go build -o $SCRIPT_DIR/macos/hello_world_macos_arm64.ext $SCRIPT_DIR
lipo -create $SCRIPT_DIR/macos/hello_world_macos_amd64.ext $SCRIPT_DIR/macos/hello_world_macos_arm64.ext -output $SCRIPT_DIR/macos/hello_world_macos.ext
rm $SCRIPT_DIR/macos/hello_world_macos_amd64.ext $SCRIPT_DIR/macos/hello_world_macos_arm64.ext

GOOS=windows GOARCH=amd64 go build -o $SCRIPT_DIR/windows/hello_world_windows.ext.exe $SCRIPT_DIR
GOOS=windows GOARCH=arm64 go build -o $SCRIPT_DIR/windows-arm64/hello_world_windows_arm64.ext.exe $SCRIPT_DIR

GOOS=linux GOARCH=amd64 go build -o $SCRIPT_DIR/linux/hello_world_linux.ext $SCRIPT_DIR
GOOS=linux GOARCH=arm64 go build -o $SCRIPT_DIR/linux-arm64/hello_world_linux_arm64.ext $SCRIPT_DIR

# 2. build hello_mars table extensions

GOOS=darwin GOARCH=amd64 go build -ldflags '-X "main.extensionName=test_extensions.hello_mars" -X "main.tableName=hello_mars" -X "main.columnValue=mars"' -o $SCRIPT_DIR/macos/hello_mars_macos_amd64.ext $SCRIPT_DIR
GOOS=darwin GOARCH=arm64 go build -ldflags '-X "main.extensionName=test_extensions.hello_mars" -X "main.tableName=hello_mars" -X "main.columnValue=mars"' -o $SCRIPT_DIR/macos/hello_mars_macos_arm64.ext $SCRIPT_DIR
lipo -create $SCRIPT_DIR/macos/hello_mars_macos_amd64.ext $SCRIPT_DIR/macos/hello_mars_macos_arm64.ext -output $SCRIPT_DIR/macos/hello_mars_macos.ext
rm $SCRIPT_DIR/macos/hello_mars_macos_amd64.ext $SCRIPT_DIR/macos/hello_mars_macos_arm64.ext

GOOS=windows GOARCH=amd64 go build -ldflags '-X "main.extensionName=test_extensions.hello_mars" -X "main.tableName=hello_mars" -X "main.columnValue=mars"' -o $SCRIPT_DIR/windows/hello_mars_windows.ext.exe $SCRIPT_DIR
GOOS=windows GOARCH=arm64 go build -ldflags '-X "main.extensionName=test_extensions.hello_mars" -X "main.tableName=hello_mars" -X "main.columnValue=mars"' -o $SCRIPT_DIR/windows-arm64/hello_mars_windows_arm64.ext.exe $SCRIPT_DIR

GOOS=linux GOARCH=amd64 go build -ldflags '-X "main.extensionName=test_extensions.hello_mars" -X "main.tableName=hello_mars" -X "main.columnValue=mars"' -o $SCRIPT_DIR/linux/hello_mars_linux.ext $SCRIPT_DIR
GOOS=linux GOARCH=arm64 go build -ldflags '-X "main.extensionName=test_extensions.hello_mars" -X "main.tableName=hello_mars" -X "main.columnValue=mars"' -o $SCRIPT_DIR/linux-arm64/hello_mars_linux_arm64.ext $SCRIPT_DIR
