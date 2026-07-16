$displayNameLike = "Dell Command*Update*"
$publisherLike = "Dell Inc.*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$ExpectedExitCodes = @(0, 1641, 3010, 1223)

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

Stop-Process -Name "DellCommandUpdate" -Force -ErrorAction SilentlyContinue
Stop-Process -Name "dcu-cli" -Force -ErrorAction SilentlyContinue

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
    if ($existingArgs -notmatch '(?i)/qn') { $existingArgs = ("$existingArgs /qn").Trim() }
    if ($existingArgs -notmatch '(?i)/norestart') { $existingArgs = ("$existingArgs /norestart").Trim() }
} else {
    if ($existingArgs -notmatch '(?i)(^|\s)/s(\s|$)') { $existingArgs = ("$existingArgs /s").Trim() }
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
