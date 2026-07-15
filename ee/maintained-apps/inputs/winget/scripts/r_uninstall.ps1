# Locates R for Windows' Inno Setup uninstaller from the registry and runs it
# silently. The DisplayName embeds the version (e.g. "R for Windows 4.6.0"), so
# match by prefix and require the R Core Team publisher.

$displayNameLike = "R for Windows*"
$publisherLike = "R Core Team*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$ExpectedExitCodes = @(0, 1641, 3010, 1223)

$selected = $null
foreach ($p in $paths) {
  $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
    $_.DisplayName -like $displayNameLike -and $_.Publisher -like $publisherLike
  }
  if ($items) { $selected = $items | Select-Object -First 1; break }
}

if (-not $selected -or (-not $selected.UninstallString -and -not $selected.QuietUninstallString)) {
  Write-Host "Uninstall entry not found"
  Exit 0
}

# Best-effort: stop R processes so the uninstaller doesn't fail on locked files.
foreach ($proc in @('Rgui', 'Rterm', 'Rscript')) {
  Stop-Process -Name $proc -Force -ErrorAction SilentlyContinue
}

$uninstallCommand = if ($selected.QuietUninstallString) { $selected.QuietUninstallString } else { $selected.UninstallString }

# Split the uninstall string into exe + args, handling quoted, unquoted-with-spaces,
# and bare-token forms.
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

# Inno Setup silent uninstall flags.
foreach ($flag in @('/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART')) {
    if ($existingArgs -notmatch [regex]::Escape($flag)) {
        $existingArgs = ("$existingArgs $flag").Trim()
    }
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
