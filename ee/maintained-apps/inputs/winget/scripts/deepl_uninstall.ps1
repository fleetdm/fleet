# Fleet extracts name from installer and saves it to PACKAGE_ID variable
$softwareName = $PACKAGE_ID

# DeepL registers its DisplayName as "DeepL"
$softwareNameLike = "DeepL"

# DeepL's silent uninstall switches (Zero Install / Inno based)
$uninstallArgs = "--uninstall --verysilent"

# DeepL may register under HKLM (machine scope) or HKCU (user scope)
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
    if ($key.DisplayName -eq $softwareNameLike) {
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

        # If the registered command already runs silently, keep its args;
        # otherwise apply the documented silent uninstall switches.
        if ($existingArgs -match '(?i)verysilent' -or $existingArgs -match '(?i)/S\b' -or $key.QuietUninstallString) {
            if ($existingArgs -ne '') { $uninstallArgs = $existingArgs }
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
