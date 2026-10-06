# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)
$timeoutSeconds = 540
$pollIntervalSeconds = 10

function Test-CitrixWorkspaceRegistered {
  [array]$uninstallKeys = Get-ChildItem `
      -Path $paths `
      -ErrorAction SilentlyContinue |
          ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue }

  foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -match '^Citrix Workspace \d{4}' `
        -and $key.Publisher -eq "Citrix Systems, Inc.") {
      return $true
    }
  }
  return $false
}

try {

$process = Start-Process -FilePath "${env:INSTALLER_PATH}" `
  -ArgumentList "/silent /norestart" `
  -PassThru

$elapsed = 0
while (-not $process.HasExited -and $elapsed -lt $timeoutSeconds) {
  Start-Sleep -Seconds $pollIntervalSeconds
  $elapsed += $pollIntervalSeconds
}

if (-not $process.HasExited) {
  Write-Host "Installer still running after ${timeoutSeconds}s"
  Exit 1
}

$exitCode = $process.ExitCode
Write-Host "Installer exited with code $exitCode after ${elapsed}s"

if ($exitCode -eq 40008) {
  Write-Host "A newer version of Citrix Workspace (LTSR or Current Release) is already installed"
}
if ($exitCode -eq 40032) {
  Write-Host "This version of Citrix Workspace is already installed"
}

# 3010: installed, reboot required.
if ($exitCode -ne 0 -and $exitCode -ne 3010 -and $exitCode -ne 40032) {
  Exit $exitCode
}

if (-not (Test-CitrixWorkspaceRegistered)) {
  Write-Host "Installer succeeded but Citrix Workspace isn't registered in Programs and Features"
  Exit 1
}

Exit 0

} catch {
  Write-Host "Error: $_"
  Exit 1
}
