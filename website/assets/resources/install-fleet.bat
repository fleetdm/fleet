@echo off
SET FLEETCTL_INSTALL_DIR=%USERPROFILE%\.fleetctl
SET FLEETCTL_BINARY_NAME=fleetctl.exe
SET NPM_API_URL=https://registry.npmjs.org/fleetctl/latest
SET ZIP_FILE=fleetctl.zip

REM Fetching the latest version from NPM's API
powershell -Command "$json = Invoke-RestMethod -Uri '%NPM_API_URL%'; $version = $json.version; Write-Host $version" > version.txt
SET /P LATEST_VERSION=<version.txt
DEL version.txt

SET DOWNLOAD_URL=https://github.com/fleetdm/fleet/releases/download/fleet-v%LATEST_VERSION%/fleetctl_v%LATEST_VERSION%_windows.zip

IF NOT EXIST "%FLEETCTL_INSTALL_DIR%" mkdir "%FLEETCTL_INSTALL_DIR%"

REM Download the zip file
powershell -Command "Invoke-WebRequest -Uri '%DOWNLOAD_URL%' -OutFile '%ZIP_FILE%'"

REM Extract the zip file
powershell -Command "Expand-Archive -Path '%ZIP_FILE%' -DestinationPath '%FLEETCTL_INSTALL_DIR%' -Force"

REM Clean up the zip file
DEL "%ZIP_FILE%"

echo Installation complete.
pause
