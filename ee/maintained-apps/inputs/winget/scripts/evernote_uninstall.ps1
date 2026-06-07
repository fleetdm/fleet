# Locates Evernote's uninstaller from the registry and runs it silently.

# Match the registry DisplayName (osquery programs.name).
$softwareNameLike = "Evernote"

# NSIS (electron-builder) silent uninstall flags. --allusers mirrors the
# machine-scope install so the per-machine entry is removed.
$silentArgs = "/S --allusers"

$paths = @(
  'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
  'HKCU:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
)

$exitCode = 0

try {

[array]$uninstallKeys = Get-ChildItem -Path $paths -ErrorAction SilentlyContinue |
    ForEach-Object { Get-ItemProperty $_.PSPath }

$foundUninstaller = $false
foreach ($key in $uninstallKeys) {
    if ($key.DisplayName -eq $softwareNameLike) {
        $foundUninstaller = $true

        # Prefer QuietUninstallString when present.
        $uninstallString = if ($key.QuietUninstallString) {
            $key.QuietUninstallString
        } else {
            $key.UninstallString
        }

        # Parse the uninstall string defensively into executable + args.
        $exePath = ""
        $existingArgs = ""
        if ($uninstallString -match '^\s*"([^"]+)"\s*(.*)$') {
            # Quoted path
            $exePath = $matches[1]
            $existingArgs = $matches[2].Trim()
        } elseif ($uninstallString -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
            # Unquoted, may contain spaces - capture through .exe
            $exePath = $matches[1]
            $existingArgs = $matches[2].Trim()
        } elseif ($uninstallString -match '^\s*(\S+)\s*(.*)$') {
            # Bare token
            $exePath = $matches[1]
            $existingArgs = $matches[2].Trim()
        } else {
            Throw "Could not parse uninstall string: $uninstallString"
        }

        # Build argument list, preserving any existing args and ensuring silent flags.
        $argumentList = @()
        if ($existingArgs -ne '') {
            $argumentList += $existingArgs -split '\s+'
        }
        if ($argumentList -notcontains '/S') {
            $argumentList += '/S'
        }
        if ($argumentList -notcontains '--allusers') {
            $argumentList += '--allusers'
        }

        Write-Host "Uninstall executable: $exePath"
        Write-Host "Uninstall arguments: $($argumentList -join ' ')"

        $processOptions = @{
            FilePath = $exePath
            ArgumentList = $argumentList
            PassThru = $true
            Wait = $true
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
