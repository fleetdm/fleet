# Uninstall Teleport Connect (NSIS / electron-builder) via its registry UninstallString.
# DisplayName is versioned (e.g. "Teleport Connect 18.8.3"), so match on the stem.
# electron-builder installers frequently register per-user, so check HKCU as well as HKLM.
# NSIS uninstaller takes /S for a silent uninstall.

$softwareNameLike = "Teleport Connect*"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall'
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
        #   "C:\path\Uninstall Teleport Connect.exe" /S   (quoted, has spaces)
        #   C:\path\Uninstall Teleport Connect.exe /S     (unquoted, has spaces)
        #   MsiExec.exe /X{GUID}                          (bare token)
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

        # Ensure the silent flag is present.
        if ($existingArgs -notmatch '(?i)(^|\s)/S(\s|$)') {
            $uninstallArgs = ("$existingArgs /S").Trim()
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
