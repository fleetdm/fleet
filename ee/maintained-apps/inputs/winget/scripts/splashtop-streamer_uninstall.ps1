# Locates Splashtop Streamer's uninstaller from the registry and runs it silently.
# Splashtop installs via an MSI, so the registry UninstallString is typically
# "MsiExec.exe /X{GUID}".

$softwareNameLike = "Splashtop Streamer*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$exitCode = 0

try {
    [array]$uninstallKeys = Get-ChildItem -Path $paths -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath }

    $foundUninstaller = $false
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

            if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
                Write-Host "Uninstall succeeded (reboot required/initiated)."
                Exit 0
            }
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
