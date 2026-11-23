# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# Splashtop's installer unpacks a per-machine MSI and runs the command it is
# given from the unpacked folder. It exits 0 whatever msiexec returns, so the
# result is read from the MSI log. A silent install closes a running Splashtop
# Business and doesn't open the app afterwards.

$exeFilePath = "${env:INSTALLER_PATH}"
$softwareName = "Splashtop Business"
$publisher = "Splashtop Inc."
$successCodes = @(0, 3010, 1641)
$timeoutSeconds = 1800

$uninstallRoots = @(
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall'
)

function Get-InstalledEntries {
  foreach ($root in $uninstallRoots) {
    Get-ChildItem -LiteralPath $root -ErrorAction SilentlyContinue | ForEach-Object {
      $item = Get-ItemProperty -LiteralPath $_.PSPath -ErrorAction SilentlyContinue
      if ($item -and $item.DisplayName -eq $softwareName -and $item.Publisher -eq $publisher) { $item }
    }
  }
}

# The installer stores its MSI product code in a settings file near its end.
function Get-InstallerProductCode {
  param([string]$Path)
  $stream = [System.IO.File]::OpenRead($Path)
  try {
    [int]$length = [Math]::Min($stream.Length, 2MB)
    $buffer = New-Object byte[] $length
    $null = $stream.Seek(-$length, [System.IO.SeekOrigin]::End)
    $read = 0
    while ($read -lt $length) {
      $count = $stream.Read($buffer, $read, $length - $read)
      if ($count -le 0) { break }
      $read += $count
    }
  } finally {
    $stream.Dispose()
  }
  $text = [System.Text.Encoding]::ASCII.GetString($buffer, 0, $read)
  if ($text -match '\[PREVERCHECK\][^\[]*?PRODUCTID=(\{[0-9A-Fa-f-]{36}\})') { return $matches[1] }
  return $null
}

# The log path is passed through the installer unquoted, so it can't contain spaces.
$logDir = [System.IO.Path]::GetTempPath()
if ($logDir -match '\s') { $logDir = Join-Path $env:SystemRoot 'Temp' }
$logFile = Join-Path $logDir ("fleet-splashtop-business-" + [guid]::NewGuid().ToString('N') + ".log")

try {

$installed = @(Get-InstalledEntries)
$productCode = Get-InstallerProductCode -Path $exeFilePath
Write-Host "Installer product code: $productCode"

# These are the properties Splashtop's own setup passes for a reinstall of the
# same build, an upgrade, and a first install.
$msiArgs = "/norestart /i setup.msi"
if ($productCode -and @($installed | Where-Object { $_.PSChildName -eq $productCode }).Count -gt 0) {
  $msiArgs += " REINSTALL=ALL REINSTALLMODE=vomus MSIENFORCEUPGRADECOMPONENTRULES=1 CA_UPGRADE=1"
} elseif ($installed.Count -gt 0) {
  $msiArgs += " CA_UPGRADE=1"
}
$msiArgs += " /qn /l*v $logFile"
Write-Host "Install command: msiexec $msiArgs"

$process = Start-Process -FilePath "$exeFilePath" -ArgumentList "msiexec $msiArgs" -PassThru
# Keeps the process object usable after the process ends.
$null = $process.Handle

if (-not $process.WaitForExit($timeoutSeconds * 1000)) {
  Write-Host "Installer did not exit within $timeoutSeconds seconds; terminating it."
  & taskkill.exe /PID $process.Id /T /F 2>&1 | Write-Host
  Exit 1603
}

$exitCode = $null
$logText = ""
if (Test-Path -LiteralPath $logFile -PathType Leaf) {
  $logText = [System.IO.File]::ReadAllText($logFile)
  $results = [regex]::Matches($logText, 'MainEngineThread is returning (\d+)')
  if ($results.Count -gt 0) { $exitCode = [int]$results[$results.Count - 1].Groups[1].Value }
}

if ($null -eq $exitCode) {
  Write-Host "The installer exited without an msiexec result, so the install can't be confirmed."
  Exit 1
}

Write-Host "Install exit code: $exitCode"
if ($successCodes -notcontains $exitCode) {
  $logText -split "`r?`n" |
    Where-Object { $_ -match 'Return value 3|Error \d{4}\.|Installation failed' } |
    Select-Object -First 20 | ForEach-Object { Write-Host $_ }
  Exit $exitCode
}

if (@(Get-InstalledEntries).Count -eq 0) {
  Write-Host "$softwareName isn't registered in Programs and Features after the install."
  Exit 1
}
Exit 0

} catch {
  Write-Host "Error: $_"
  Exit 1
} finally {
  Remove-Item -LiteralPath $logFile -Force -ErrorAction SilentlyContinue
}
