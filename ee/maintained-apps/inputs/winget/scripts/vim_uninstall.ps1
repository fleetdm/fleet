# Locates Vim's NsisMultiUser uninstaller from the registry and runs it
# silently. NSIS matches the literal " _?=" and takes the rest of the command
# line verbatim, so the directory must be unquoted or is_valid_instpath()
# rejects it and the uninstaller exits 2 before doing any work.

$displayNameLike = "Vim *"
$publisherLike = "The Vim Project*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$entry = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -like $displayNameLike -and $_.Publisher -like $publisherLike
  }
  if ($items) { $entry = $items | Select-Object -First 1; break }
}

if (-not $entry -or (-not $entry.UninstallString -and -not $entry.QuietUninstallString)) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

Stop-Process -Name "gvim" -Force -ErrorAction SilentlyContinue
Stop-Process -Name "vim" -Force -ErrorAction SilentlyContinue

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

if ($existingArgs -notmatch '(?i)(^|\s)/S(\s|$)') { $existingArgs = ("$existingArgs /S").Trim() }
if ($entry.InstallLocation -and ($existingArgs -notmatch '_\?=')) {
    $installDir = $entry.InstallLocation.TrimEnd('\')
    $existingArgs = ("$existingArgs _?=$installDir").Trim()
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

    # The in-place uninstaller cannot delete its own directory. What's left is
    # the uninstaller and the installer-written default _vimrc; per-user vimrc
    # files in user profiles are not touched.
    if ($exitCode -eq 0 -and $entry.InstallLocation -and (Test-Path $entry.InstallLocation)) {
        Remove-Item -LiteralPath $entry.InstallLocation.TrimEnd('\') -Recurse -Force -ErrorAction SilentlyContinue
    }
    Exit $exitCode
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
