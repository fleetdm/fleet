# Fleet extracts name from installer and saves it to PACKAGE_ID variable
$softwareName = $PACKAGE_ID

# Citrix Workspace registers a version-specific DisplayName (e.g. "Citrix Workspace 2603").
# Match the base name with a wildcard.
$softwareNameLike = "*Citrix Workspace*"

# Citrix uninstaller (TrolleyExpress.exe) silent uninstall switches
$uninstallArgs = "/uninstall /cleanup /silent /noreboot"

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
    # Exclude the (USB) and (DV) component sub-entries; target the main app
    if ($key.DisplayName -like $softwareNameLike -and `
        $key.DisplayName -notlike "*(USB)*" -and `
        $key.DisplayName -notlike "*(DV)*") {
        $foundUninstaller = $true

        # Prefer QuietUninstallString when present
        $uninstallCommand = if ($key.QuietUninstallString) {
            $key.QuietUninstallString
        } else {
            $key.UninstallString
        }

        # Parse the uninstall string defensively into command + args
        $u = $uninstallCommand
        if ($u -match '^\s*"([^"]+)"\s*(.*)$') {
            # Quoted path
            $uninstallCommand = $matches[1]
            $existingArgs = $matches[2].Trim()
        } elseif ($u -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
            # Unquoted path that may contain spaces — capture through .exe
            $uninstallCommand = $matches[1]
            $existingArgs = $matches[2].Trim()
        } else {
            # Bare token
            $parts = $u -split '\s+', 2
            $uninstallCommand = $parts[0]
            $existingArgs = if ($parts.Length -gt 1) { $parts[1].Trim() } else { "" }
        }

        # If the existing args already request a silent uninstall, use them as-is;
        # otherwise apply Citrix's documented silent switches.
        if ($existingArgs -match '(?i)/silent' -or $existingArgs -match '(?i)/uninstall') {
            $uninstallArgs = $existingArgs
        }

        Write-Host "Uninstall command: $uninstallCommand"
        Write-Host "Uninstall args: $uninstallArgs"

        $processOptions = @{
            FilePath = $uninstallCommand
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

Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
