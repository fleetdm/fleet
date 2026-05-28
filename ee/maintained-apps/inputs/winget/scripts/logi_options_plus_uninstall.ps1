# Locates Logi Options+ from the registry and runs its uninstaller silently.

$softwareNameLike = "*Logi Options*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$exitCode = 0

try {

[array]$uninstallKeys = Get-ChildItem `
    -Path $paths `
    -ErrorAction SilentlyContinue |
        ForEach-Object { Get-ItemProperty $_.PSPath }

$foundUninstaller = $false
foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -like $softwareNameLike -and $key.Publisher -like "*Logitech*") {
        $foundUninstaller = $true
        $uninstallCommand = if ($key.QuietUninstallString) {
            $key.QuietUninstallString
        } else {
            $key.UninstallString
        }

        # Split the uninstall string into exe + args. Handle both quoted and
        # unquoted exe paths.
        $exePath = ""
        $existingArgs = ""
        if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
            $exePath = $matches[1]
            $existingArgs = $matches[2].Trim()
        } elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
            $exePath = $matches[1]
            $existingArgs = $matches[2].Trim()
        } else {
            Throw "Could not parse uninstall string: $uninstallCommand"
        }

        if ($existingArgs -notmatch '/quiet' -and $existingArgs -notmatch '/qn') {
            $uninstallArgs = ("$existingArgs /quiet").Trim()
        } else {
            $uninstallArgs = $existingArgs
        }

        Write-Host "Uninstall command: $exePath"
        Write-Host "Uninstall args: $uninstallArgs"

        $processOptions = @{
            FilePath = $exePath
            PassThru = $true
            Wait = $true
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
    Exit 1
}

# Logitech's installer reports success with exit code 0 or -1978335226
# (per the winget manifest's InstallerSuccessCodes). msiexec also returns
# 3010/1641 on reboot-required success.
if ($exitCode -eq 0 -or $exitCode -eq -1978335226 -or $exitCode -eq 3010 -or $exitCode -eq 1641) {
    Exit 0
}

Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
