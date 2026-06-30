$displayNameLike = "Remote Help*"
$publisherLike = "Microsoft Corporation*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$ExpectedExitCodes = @(0, 1641, 3010, 1223)

# A WiX Burn bundle and its child MSI packages can all share the same DisplayName.
# The bundle is the entry that exposes a QuietUninstallString / an .exe bootstrapper
# UninstallString; child MSIs only expose "MsiExec.exe /I{GUID}". Prefer the bundle
# so the whole product is removed (uninstalling a single child MSI would not).
$candidates = @()
foreach ($p in $paths) {
  $candidates += Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -like $displayNameLike -and $_.Publisher -like $publisherLike
  }
}

if (-not $candidates -or $candidates.Count -eq 0) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

$entry = $candidates | Where-Object { $_.QuietUninstallString } | Select-Object -First 1
if (-not $entry) {
  $entry = $candidates | Where-Object { $_.UninstallString -and $_.UninstallString -notmatch '(?i)msiexec' } | Select-Object -First 1
}
if (-not $entry) {
  $entry = $candidates | Where-Object { $_.UninstallString } | Select-Object -First 1
}
if (-not $entry) {
  Write-Host "Uninstall entry found but has no uninstall string"
  Exit 0
}

Stop-Process -Name "RemoteHelp" -Force -ErrorAction SilentlyContinue

$uninstallCommand = if ($entry.QuietUninstallString) { $entry.QuietUninstallString } else { $entry.UninstallString }

$exePath = ""
$existingArgs = ""
if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
    $exePath = $matches[1]; $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    $exePath = $matches[1]; $existingArgs = $matches[2].Trim()
} elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
    $exePath = $matches[1]; $existingArgs = $matches[2].Trim()
} else {
    Throw "Could not parse uninstall string: $uninstallCommand"
}

if ($exePath -match '(?i)msiexec') {
    # Force an uninstall (/x), never a repair (/i), and run with MSI quiet flags.
    $existingArgs = $existingArgs -replace '(?i)/i(\{)', '/x$1'
    $existingArgs = ($existingArgs -replace '(?i)/uninstall', '') -replace '(?i)/quiet', ''
    if ($existingArgs -notmatch '(?i)/x') { $existingArgs = ("/x $existingArgs").Trim() }
    if ($existingArgs -notmatch '(?i)/qn') { $existingArgs = ("$existingArgs /qn").Trim() }
    if ($existingArgs -notmatch '(?i)/norestart') { $existingArgs = ("$existingArgs /norestart").Trim() }
} else {
    # WiX Burn bootstrapper.
    if ($existingArgs -notmatch '(?i)/uninstall') { $existingArgs = ("$existingArgs /uninstall").Trim() }
    if ($existingArgs -notmatch '(?i)/quiet') { $existingArgs = ("$existingArgs /quiet").Trim() }
    if ($existingArgs -notmatch '(?i)/norestart') { $existingArgs = ("$existingArgs /norestart").Trim() }
}

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
