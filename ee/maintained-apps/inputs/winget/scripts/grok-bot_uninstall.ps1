$displayNameLike = "Grok Bot *"
$publisher = "SpaceXAI"

$paths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
    foreach ($sub in (Get-ChildItem -Path $p -ErrorAction SilentlyContinue)) {
        $key = Get-ItemProperty $sub.PSPath -ErrorAction SilentlyContinue
        if ($key.DisplayName -like $displayNameLike -and $key.Publisher -eq $publisher) {
            $uninstall = $key
            break
        }
    }
    if ($uninstall) { break }
}

if (-not $uninstall -or (-not $uninstall.UninstallString -and -not $uninstall.QuietUninstallString)) {
    Write-Host "Grok Bot is not installed."
    Exit 0
}

$uninstallCommand = if ($uninstall.QuietUninstallString) { $uninstall.QuietUninstallString } else { $uninstall.UninstallString }

$exePath = ""
$arguments = ""
if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
    $exePath = $matches[1]
    $arguments = $matches[2].Trim()
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    $exePath = $matches[1]
    $arguments = $matches[2].Trim()
} else {
    Write-Host "Error: Could not parse uninstall string: $uninstallCommand"
    Exit 1
}

# \b does not match between a space and a slash, so anchor on whitespace.
if ($arguments -notmatch '(?i)(^|\s)/S($|\s)') { $arguments = "$arguments /S".Trim() }
if ($arguments -notmatch '(?i)(^|\s)/allusers($|\s)') { $arguments = "$arguments /allusers".Trim() }

$installDir = Split-Path $exePath -Parent

# Stop only the copy in this install directory.
Get-Process -ErrorAction SilentlyContinue |
    Where-Object { $_.Path -and $_.Path.StartsWith("$installDir\", [System.StringComparison]::OrdinalIgnoreCase) } |
    Stop-Process -Force -ErrorAction SilentlyContinue

Write-Host "Uninstall command: $exePath"
Write-Host "Uninstall args: $arguments"

try {
    $process = Start-Process -FilePath $exePath -ArgumentList $arguments -NoNewWindow -PassThru -Wait
    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"
    if ($exitCode -ne 0) { Exit $exitCode }

    # The NSIS uninstaller relaunches itself from %TEMP%, so the process above exits
    # while removal is still in flight.
    $keyPath = $uninstall.PSPath
    for ($waited = 0; $waited -lt 120; $waited++) {
        if (-not (Test-Path -LiteralPath $keyPath) -and -not (Test-Path -LiteralPath $exePath)) { break }
        Start-Sleep -Seconds 1
    }

    if (Test-Path -LiteralPath $keyPath) {
        Write-Host "Error: Grok Bot is still registered after uninstalling."
        Exit 1
    }

    Write-Host "Grok Bot removed."
    Exit 0
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
