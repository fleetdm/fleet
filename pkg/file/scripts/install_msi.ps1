Start-Process 'msiexec.exe' -ArgumentList /a /lv /norestart "${INSTALLER_PATH}" -Wait -NoNewWindow
