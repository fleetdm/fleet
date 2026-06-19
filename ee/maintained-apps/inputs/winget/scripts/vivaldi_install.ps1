# Install Vivaldi silently, machine-wide (Chromium-based browser).
# Fleet runs installs as SYSTEM, so --system-level is required to install for
# all users under %ProgramFiles% (and register under HKLM). Without it the
# installer lands in the SYSTEM profile and is invisible to the real user.
$process = Start-Process -FilePath $env:INSTALLER_PATH `
  -ArgumentList "--vivaldi-silent --do-not-launch-chrome --system-level" `
  -NoNewWindow -PassThru -Wait
Exit $process.ExitCode
