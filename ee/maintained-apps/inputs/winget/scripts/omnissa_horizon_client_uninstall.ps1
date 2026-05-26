# Locates Omnissa Horizon Client's uninstaller from the registry and runs it
# silently. The Horizon Client ships as a WiX Burn bundle, but the registered
# UninstallString is an unquoted "MsiExec.exe /X{ProductCode}" — so the parser
# below handles both quoted and unquoted exe paths.

$softwareNameLike = "*Omnissa Horizon Client*"

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
    if ($key.DisplayName -like $softwareNameLike) {
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

        # Append /quiet /norestart if the registered command doesn't already
        # request silent execution.
        if ($existingArgs -notmatch '/quiet' -and $existingArgs -notmatch '/qn') {
            $uninstallArgs = ("$existingArgs /quiet /norestart").Trim()
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
    Exit 0
}

# msiexec returns 3010 (ERROR_SUCCESS_REBOOT_REQUIRED) or 1641
# (ERROR_SUCCESS_REBOOT_INITIATED) on successful uninstall when a reboot is
# needed to finish. Treat both as success.
if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
    Exit 0
}

Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
