# Install Vivaldi silently (user-scoped Chromium-based browser)
$process = Start-Process -FilePath $env:INSTALLER_PATH `
  -ArgumentList "--vivaldi-silent --do-not-launch-chrome" `
  -NoNewWindow -PassThru -Wait
Exit $process.ExitCode
