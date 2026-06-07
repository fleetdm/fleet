# Uninstalls Splashtop Streamer.
#
# Splashtop Streamer installs via an embedded MSI with a stable ProductCode
# (per the winget manifest: {B7C5EA94-B96A-41F5-BE95-25D78B486678}). We try
# that ProductCode first (most reliable), then fall back to locating the
# uninstaller from the registry by DisplayName. Treat 0/3010/1641 as success.

$productCode = "{B7C5EA94-B96A-41F5-BE95-25D78B486678}"
$softwareNameLike = "Splashtop Streamer*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

function Test-Success([int]$code) {
    return ($code -eq 0 -or $code -eq 3010 -or $code -eq 1641)
}

try {
    # 1) ProductCode-based uninstall if that key is registered.
    $pcKeyExists = $false
    foreach ($p in $paths) {
        if (Test-Path (Join-Path $p $productCode)) { $pcKeyExists = $true; break }
    }
    if ($pcKeyExists) {
        $args = "/x $productCode /qn /norestart"
        Write-Host "Uninstall command: msiexec.exe"
        Write-Host "Uninstall args: $args"
        $process = Start-Process -FilePath "msiexec.exe" -ArgumentList $args -PassThru -Wait -NoNewWindow
        $exitCode = $process.ExitCode
        Write-Host "Uninstall exit code: $exitCode"
        if (Test-Success $exitCode) { Exit 0 }
        Exit $exitCode
    }

    # 2) Fall back to registry DisplayName lookup.
    [array]$uninstallKeys = Get-ChildItem -Path $paths -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath }

    $foundUninstaller = $false
    $exitCode = 0
    foreach ($key in $uninstallKeys) {
        if ($key.DisplayName -like $softwareNameLike) {
            $foundUninstaller = $true

            $uninstallCommand = if ($key.QuietUninstallString) {
                $key.QuietUninstallString
            } else {
                $key.UninstallString
            }

            # Parse the UninstallString defensively (quoted / unquoted-with-spaces / bare MsiExec).
            $exe = $null
            $existingArgs = ""
            if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
                $exe = $matches[1]
                $existingArgs = $matches[2].Trim()
            } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
                $exe = $matches[1]
                $existingArgs = $matches[2].Trim()
            } elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
                $exe = $matches[1]
                $existingArgs = $matches[2].Trim()
            } else {
                Write-Host "Error: could not parse uninstall command: $uninstallCommand"
                Exit 1
            }

            if ($exe -match '(?i)msiexec') {
                $existingArgs = $existingArgs -replace '(?i)/I', '/X'
                if ($existingArgs -notmatch '(?i)/quiet|/qn') { $existingArgs = "$existingArgs /qn" }
                if ($existingArgs -notmatch '(?i)/norestart') { $existingArgs = "$existingArgs /norestart" }
                $uninstallArgs = $existingArgs.Trim()
            } else {
                $uninstallArgs = $existingArgs.Trim()
            }

            Write-Host "Uninstall command: $exe"
            Write-Host "Uninstall args: $uninstallArgs"

            $process = Start-Process -FilePath $exe -ArgumentList $uninstallArgs -PassThru -Wait -NoNewWindow
            $exitCode = $process.ExitCode
            Write-Host "Uninstall exit code: $exitCode"

            if (Test-Success $exitCode) { Exit 0 }
            break
        }
    }

    if (-not $foundUninstaller) {
        Write-Host "Uninstall entry not found for $softwareNameLike"
        Exit 0
    }

    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
