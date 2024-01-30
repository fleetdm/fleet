@echo off
SET FLEETCTL_INSTALL_DIR=%USERPROFILE%\.fleetctl
SET FLEETCTL_BINARY_NAME=fleetctl.exe
SET FLEETCTL_REPO_URL=https://raw.githubusercontent.com/fleetdm/fleet/main/tools/fleetctl-npm/package.json
SET ZIP_FILE=fleetctl.zip

powershell -Command "(New-Object Net.WebClient).DownloadFile('%FLEETCTL_REPO_URL%', 'package.json')"
FOR /F "tokens=2 delims=:" %%i IN ('findstr version package.json') DO SET LATEST_VERSION=%%i
SET LATEST_VERSION=%LATEST_VERSION:"=%
SET LATEST_VERSION=%LATEST_VERSION:,=%
SET LATEST_VERSION=%LATEST_VERSION:~1%

SET DOWNLOAD_URL=https://github.com/fleetdm/fleet/releases/download/fleet-%LATEST_VERSION%/fleetctl_%LATEST_VERSION%_windows.zip

IF NOT EXIST "%FLEETCTL_INSTALL_DIR%" mkdir "%FLEETCTL_INSTALL_DIR%"

REM Download the zip file
powershell -Command "Invoke-WebRequest -Uri '%DOWNLOAD_URL%' -OutFile '%ZIP_FILE%'"

REM Extract the zip file
powershell -Command "Expand-Archive -Path '%ZIP_FILE%' -DestinationPath '%FLEETCTL_INSTALL_DIR%'"

REM Clean up the zip file
DEL "%ZIP_FILE%"

REM Temporary PATH update for current session
SET PATH=%PATH%;%FLEETCTL_INSTALL_DIR%

echo Installation complete.
echo fleetctl was added to your PATH for this session.
echo To use fleetctl, ensure %FLEETCTL_INSTALL_DIR% is in your PATH.
pause
