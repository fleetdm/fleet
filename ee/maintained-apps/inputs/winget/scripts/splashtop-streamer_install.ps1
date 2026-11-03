# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Splashtop's installer wraps a per-machine MSI and uses its own switches.
# To link the streamer to a Splashtop Business account, add your deployment
# code to the parameter list: "hidewindow=1,dcode=XXXXXXXXXXXX,confirm_d=0".

$exeFilePath = "${env:INSTALLER_PATH}"
$softwareName = "Splashtop Streamer"
$successCodes = @(0, 3010, 1641)

$uninstallPaths = @(
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*',
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\*'
)

function Test-StreamerRegistered {
  $entry = Get-ItemProperty -Path $uninstallPaths -ErrorAction SilentlyContinue |
    Where-Object { $_.DisplayName -eq $softwareName -and $_.DisplayVersion } |
    Select-Object -First 1
  if (-not $entry) { return $false }
  $installed = $null
  if (-not [version]::TryParse($entry.DisplayVersion, [ref]$installed)) { return $false }
  return $installed -ge $targetVersion
}

try {

# An older Streamer stays registered until the inner MSI replaces it, so wait
# for at least the installer's own version.
$vi = (Get-Item -LiteralPath $exeFilePath).VersionInfo
$targetVersion = [version]::new($vi.ProductMajorPart, $vi.ProductMinorPart, $vi.ProductBuildPart, $vi.ProductPrivatePart)
Write-Host "Installer version: $targetVersion"

$process = Start-Process -FilePath "$exeFilePath" `
  -ArgumentList "prevercheck /s /i hidewindow=1" `
  -PassThru -Wait
$exitCode = $process.ExitCode
Write-Host "Install exit code: $exitCode"

if ($successCodes -notcontains $exitCode) { Exit $exitCode }

# The wrapper can exit before the inner MSI finishes registering.
$deadline = (Get-Date).AddSeconds(180)
while ((Get-Date) -lt $deadline) {
  if (Test-StreamerRegistered) {
    Write-Host "$softwareName $targetVersion or later is registered."
    Exit 0
  }
  Start-Sleep -Seconds 5
}

Write-Host "Timed out waiting for $softwareName $targetVersion to register in Programs and Features."
Exit 1

} catch {
  Write-Host "Error: $_"
  Exit 1
}
