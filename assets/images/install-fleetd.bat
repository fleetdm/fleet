@echo off
echo Downloading fleetd-base.msi...
powershell -Command "(New-Object Net.WebClient).DownloadFile('https://download.fleetdm.com/fleetd-base.msi', 'fleetd-base.msi')"

echo Installing fleetd-base.msi...
msiexec /i fleetd-base.msi FLEET_URL="https://5ce2-2603-8081-7703-92-b186-b700-91f9-e4a.ngrok-free.app" FLEET_SECRET="QCwNzf0eMwJ9QaWTlh8JR9BdK/qv3ngx"

echo Installation complete.
exit