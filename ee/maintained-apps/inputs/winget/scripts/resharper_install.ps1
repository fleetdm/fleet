# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# ReSharper is a Visual Studio extension. The installer is a web bootstrapper that
# downloads the product packages, so this can take a while. /PerMachine=True keeps
# the install out of the SYSTEM profile, /SkipEtwService=True avoids the UAC prompt
# JetBrains documents as unavoidable, and /VsVersion names the Visual Studio
# instances to integrate into.
# https://resharper-support.jetbrains.com/hc/en-us/articles/207241485

$exeFilePath = "${env:INSTALLER_PATH}"

# Overall cap stays under production's 1 hour limit for install scripts. The wait
# below gives up as soon as the installer is no longer running, so a failed
# install reports its log well inside the validator's 10 minute script cap.
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

# Wait for the uninstall registry entry, which is what osquery reports. The
# bootstrapper can outlive its own exit code, so keep waiting while an installer
# process is still running and give up once none is left.
$deadline = (Get-Date).AddSeconds($registryTimeoutSeconds)
$entries = @(Get-ReSharperEntries)
while ($entries.Count -eq 0 -and (Get-Date) -lt $deadline) {
  $installerRunning = @(Get-Process -Name 'JetBrains.Platform.Installer*' -ErrorAction SilentlyContinue).Count -gt 0
  Start-Sleep -Seconds 10
  $entries = @(Get-ReSharperEntries)
  if ($entries.Count -eq 0 -and -not $installerRunning) { break }
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
