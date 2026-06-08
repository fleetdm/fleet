# Locates Git for Windows' Inno Setup uninstaller from the registry and runs it
# silently. The registry DisplayName is not reliably "Git version <ver>" (e.g. the
# GitHub-hosted runner's Git is listed as just "Git"), so anchor the match on the
# publisher -- which is unique to Git for Windows -- and only loosely guard the
# DisplayName. This mirrors the generated exists query's publisher clause.

$displayNameLike = "Git*"
$publisherLike = "The Git Development Community*"

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

# Best-effort: stop Git-related processes so the uninstaller doesn't fail on locked files.
foreach ($proc in @('git', 'bash', 'sh', 'ssh-agent', 'gitk', 'wish')) {
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

    # Inno Setup's unins000.exe relaunches itself (it copies to a temp _iu*.tmp and
    # spawns that copy), so the process we waited on returns BEFORE the uninstall
    # has finished. Poll the registry until the entry is gone so the post-uninstall
    # state is consistent for callers (e.g. Fleet's install/uninstall verification).
    $deadline = (Get-Date).AddSeconds(120)
    do {
        Start-Sleep -Seconds 2
        $stillPresent = $false
        foreach ($p in $paths) {
            $match = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
                $_.DisplayName -like $displayNameLike -and $_.Publisher -like $publisherLike
            }
            if ($match) { $stillPresent = $true; break }
        }
    } while ($stillPresent -and ((Get-Date) -lt $deadline))

    if ($stillPresent) {
        Write-Host "Uninstall entry still present after waiting; uninstall did not complete"
        Exit 1
    }

    if ($ExpectedExitCodes -contains $exitCode) { Exit 0 }
    Exit $exitCode
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
