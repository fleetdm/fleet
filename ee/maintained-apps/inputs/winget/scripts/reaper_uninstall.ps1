# Locates REAPER's NSIS uninstaller from the registry and runs it silently.

$softwareNameLike = "REAPER*"

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

            # Prefer QuietUninstallString when present.
            $uninstallCommand = if ($key.QuietUninstallString) {
                $key.QuietUninstallString
            } else {
                $key.UninstallString
            }

            # Parse the UninstallString defensively. It can be:
            #   "C:\path with spaces\uninstall.exe" /S      (quoted)
            #   C:\path with spaces\uninstall.exe /S        (unquoted, has spaces)
            #   MsiExec.exe /X{GUID}                        (bare token)
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

            # Ensure silent flag is present.
            if ($existingArgs -notmatch '(?i)\B/S\b') {
                $uninstallArgs = ("$existingArgs /S").Trim()
            } else {
                $uninstallArgs = $existingArgs
            }

            Write-Host "Uninstall command: $exe"
            Write-Host "Uninstall args: $uninstallArgs"

            $processOptions = @{
                FilePath = $exe
                PassThru = $true
                Wait = $true
                NoNewWindow = $true
            }
            if ($uninstallArgs -ne '') {
                $processOptions.ArgumentList = $uninstallArgs
            }

            $process = Start-Process @processOptions
            $exitCode = $process.ExitCode
            Write-Host "Uninstall exit code: $exitCode"
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
