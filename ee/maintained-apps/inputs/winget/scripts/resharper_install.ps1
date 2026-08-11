# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# ReSharper is a Visual Studio extension installed by a web bootstrapper.
# /PerMachine=True keeps it out of the SYSTEM profile, /SkipEtwService=True avoids
# an unavoidable UAC prompt, /VsVersion picks the VS instances to integrate into.
# https://resharper-support.jetbrains.com/hc/en-us/articles/207241485

$exeFilePath = "${env:INSTALLER_PATH}"

$registryTimeoutSeconds = 3300

$uninstallPaths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

function Get-ReSharperEntries {
  Get-ChildItem -Path $uninstallPaths -ErrorAction SilentlyContinue |
    ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
    Where-Object {
      $_.DisplayName -like 'JetBrains ReSharper*' -and
      $_.DisplayName -notlike 'JetBrains ReSharper C++*' -and
      $_.DisplayName -notlike 'JetBrains ReSharper SDK*' -and
      $_.Publisher -like '*JetBrains*'
    }
}

try {

$vsWhere = Join-Path ${env:ProgramFiles(x86)} 'Microsoft Visual Studio\Installer\vswhere.exe'
if (-not (Test-Path $vsWhere)) {
  Write-Host "Visual Studio Installer not found at $vsWhere."
  Write-Host "ReSharper is a Visual Studio extension and cannot be installed without Visual Studio."
  Exit 1
}

$installationVersions = & $vsWhere -all -prerelease -products '*' -property installationVersion 2>$null
$vsMajors = @(
  $installationVersions |
    Where-Object { $_ -match '^\d+' } |
    ForEach-Object { [int](($_ -split '\.')[0]) } |
    Sort-Object -Unique -Descending
)

if ($vsMajors.Count -eq 0) {
  Write-Host "No Visual Studio instances were reported by vswhere."
  Write-Host "ReSharper is a Visual Studio extension and cannot be installed without Visual Studio."
  Exit 1
}

$vsVersions = ($vsMajors | ForEach-Object { "$_.0" }) -join ';'
Write-Host "Visual Studio instances detected: $vsVersions"

$logFile = Join-Path $env:TEMP 'resharper-install.log'

$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/Silent=True /PerMachine=True /SkipEtwService=True /VsVersion=$vsVersions /LogFile=`"$logFile`""
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode
Write-Host "Installer exit code: $exitCode"

$deadline = (Get-Date).AddSeconds($registryTimeoutSeconds)
$entries = @(Get-ReSharperEntries)
while ($entries.Count -eq 0 -and (Get-Date) -lt $deadline) {
  $installerWasRunning = @(Get-Process -Name 'JetBrains.Platform.Installer*' -ErrorAction SilentlyContinue).Count -gt 0
  Start-Sleep -Seconds 10
  $entries = @(Get-ReSharperEntries)
  $installerIsRunning = @(Get-Process -Name 'JetBrains.Platform.Installer*' -ErrorAction SilentlyContinue).Count -gt 0
  if ($entries.Count -eq 0 -and -not $installerWasRunning -and -not $installerIsRunning) { break }
}

if ($entries.Count -eq 0) {
  Write-Host "The ReSharper uninstall registry entry did not appear."
  if (Test-Path $logFile) {
    Write-Host "--- last 50 lines of $logFile ---"
    Get-Content $logFile -Tail 50 | ForEach-Object { Write-Host $_ }
  }
  if ($exitCode -eq 0) { Exit 1 }
  Exit $exitCode
}

foreach ($entry in $entries) {
  Write-Host "Installed: DisplayName='$($entry.DisplayName)' DisplayVersion='$($entry.DisplayVersion)' Publisher='$($entry.Publisher)'"
}

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
