# SSMS 22 is owned by the Visual Studio Installer. Its ARP entry registers under
# DisplayName "SQL Server Management Studio 22" (verified via osquery on a real
# host). The UninstallString points at the VS Installer's setup.exe with an
# "uninstall --installPath ..." command; we look it up from the registry rather
# than hard-coding the install path, then ensure the silent switches are present.
$displayName = "SQL Server Management Studio 22"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$ExpectedExitCodes = @(0, 1641, 3010)

$uninstall = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -eq $displayName
  }
  if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall -or (-not $uninstall.UninstallString -and -not $uninstall.QuietUninstallString)) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

$uninstallCommand = if ($uninstall.QuietUninstallString) {
    $uninstall.QuietUninstallString
} else {
    $uninstall.UninstallString
}

$exePath = ""
$existingArgs = ""
if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
    $exePath = $matches[1]
    $existingArgs = $matches[2].Trim()
} else {
    Throw "Could not parse uninstall string: $uninstallCommand"
}

# Ensure the VS Installer runs the uninstall verb silently and waits.
if ($existingArgs -notmatch '(?i)\buninstall\b') { $existingArgs = ("uninstall $existingArgs").Trim() }
if ($existingArgs -notmatch '(?i)--quiet')      { $existingArgs = ("$existingArgs --quiet").Trim() }
if ($existingArgs -notmatch '(?i)--norestart')  { $existingArgs = ("$existingArgs --norestart").Trim() }

Write-Host "Uninstall command: $exePath"
Write-Host "Uninstall args: $existingArgs"

try {
    $processOptions = @{
        FilePath = $exePath
        ArgumentList = $existingArgs
        NoNewWindow = $true
        PassThru = $true
        Wait = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"
    if ($ExpectedExitCodes -contains $exitCode) { Exit 0 }
    Exit $exitCode
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
