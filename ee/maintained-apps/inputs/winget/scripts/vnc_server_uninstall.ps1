# Locate VNC Server (RealVNC) in the uninstall registry by DisplayName + Publisher and
# run its MSI uninstaller silently. The DisplayName embeds the marketing version
# (e.g. "RealVNC Server 7.17.0"), so match by prefix. Publisher is "RealVNC".

$displayNamePattern = "RealVNC Server *"
$publisher = "RealVNC"

$paths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
    $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
        $_.DisplayName -and ($_.Publisher -eq $publisher) -and ($_.DisplayName -like $displayNamePattern)
    }
    if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall -or -not $uninstall.UninstallString) {
    Write-Host "Uninstall entry not found"
    Exit 0
}

$uninstallCommand = $uninstall.UninstallString

# MSI products: UninstallString is "MsiExec.exe /I{GUID}" or "/X{GUID}" -- force a silent /X.
if ($uninstallCommand -match "MsiExec\.exe\s+/[IX]\s*(\{[A-Fa-f0-9-]+\})") {
    $productCode = $Matches[1]
    $uninstallCommand = "MsiExec.exe"
    $uninstallArgs = "/X $productCode /qn /norestart"
} elseif ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
    # Quoted path, possibly with trailing args.
    $uninstallCommand = $Matches[1]
    $uninstallArgs = "$($Matches[2]) /S".Trim()
} elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
    # Unquoted path that may contain spaces -- capture through .exe.
    $uninstallCommand = $Matches[1]
    $uninstallArgs = "$($Matches[2]) /S".Trim()
} else {
    Write-Host "Error: Unable to parse uninstall command: $uninstallCommand"
    Exit 1
}

Write-Host "Uninstall command: $uninstallCommand"
Write-Host "Uninstall args: $uninstallArgs"

try {
    $process = Start-Process -FilePath $uninstallCommand -ArgumentList $uninstallArgs -NoNewWindow -PassThru -Wait
    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"

    # 3010 = success, reboot required; 1641 = success, reboot initiated.
    if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
        Exit 0
    }

    Exit $exitCode
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
