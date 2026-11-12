# Uninstall Viscosity (Inno Setup) by locating its registry UninstallString.
# Inno's uninstaller (unins000.exe) takes /VERYSILENT for a silent removal. The
# registry DisplayName is versioned (e.g. "Viscosity 1.12 (1857) ..."), so match
# on the "Viscosity" stem.

$softwareNameLike = "Viscosity*"

# Inno Setup silent uninstall switches
$silentArgs = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART"

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

        # Prefer QuietUninstallString when present (already includes silent flags).
        $uninstallString = if ($key.QuietUninstallString) {
            $key.QuietUninstallString
        } else {
            $key.UninstallString
        }

        if (-not $uninstallString) {
            Write-Host "No uninstall string found for $($key.DisplayName)"
            continue
        }

        # Parse the uninstall string defensively into command + args:
        #   "C:\path with spaces\unins000.exe" /VERYSILENT   (quoted)
        #   C:\path\unins000.exe /VERYSILENT                 (unquoted, may have spaces)
        #   MsiExec.exe /X{GUID}                             (bare token)
        if ($uninstallString -match '^\s*"([^"]+)"\s*(.*)$') {
            $uninstallCommand = $Matches[1]
            $existingArgs = $Matches[2].Trim()
        } elseif ($uninstallString -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
            $uninstallCommand = $Matches[1]
            $existingArgs = $Matches[2].Trim()
        } else {
            $parts = $uninstallString -split '\s+', 2
            $uninstallCommand = $parts[0]
            $existingArgs = if ($parts.Length -gt 1) { $parts[1].Trim() } else { "" }
        }

        # Ensure silent switches are present.
        if ($existingArgs -notmatch '(?i)/VERYSILENT|/SILENT') {
            $uninstallArgs = ("$existingArgs $silentArgs").Trim()
        } else {
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
