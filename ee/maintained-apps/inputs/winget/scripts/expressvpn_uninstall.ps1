# Locates ExpressVPN's WiX burn bundle uninstaller from the registry and runs it silently.

# Match the registry DisplayName (osquery programs.name).
$softwareNameLike = "ExpressVPN"

# WiX burn bundle silent uninstall flags.
$silentArgs = "/uninstall /quiet /norestart"

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

        # Burn bundles expose the bundle .exe via UninstallString; pass burn flags.
        $uninstallString = $key.UninstallString

        # Parse the uninstall string defensively into executable + args.
        $exePath = ""
        $existingArgs = ""
        if ($uninstallString -match '^\s*"([^"]+)"\s*(.*)$') {
            $exePath = $matches[1]
            $existingArgs = $matches[2].Trim()
        } elseif ($uninstallString -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
            $exePath = $matches[1]
            $existingArgs = $matches[2].Trim()
        } elseif ($uninstallString -match '^\s*(\S+)\s*(.*)$') {
            $exePath = $matches[1]
            $existingArgs = $matches[2].Trim()
        } else {
            Throw "Could not parse uninstall string: $uninstallString"
        }

        # Build argument list ensuring the burn silent uninstall flags are present.
        $argumentList = @()
        if ($existingArgs -ne '') {
            $argumentList += $existingArgs -split '\s+'
        }
        if ($argumentList -notcontains '/uninstall') {
            $argumentList += '/uninstall'
        }
        if ($argumentList -notcontains '/quiet') {
            $argumentList += '/quiet'
        }
        if ($argumentList -notcontains '/norestart') {
            $argumentList += '/norestart'
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
