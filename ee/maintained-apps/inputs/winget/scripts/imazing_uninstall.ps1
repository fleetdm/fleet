# Locates iMazing's Inno Setup uninstaller from the registry and runs it silently.

# Match the registry DisplayName (osquery programs.name).
$softwareNameLike = "iMazing"

# Inno Setup silent uninstall flags. /NORESTART suppresses the shell-extension reboot.
$silentArgs = "/VERYSILENT /SUPPRESSMSGBOXES /NORESTART /CLOSEAPPLICATIONS"

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

        # Build argument list ensuring the Inno silent uninstall flags are present.
        $argumentList = @()
        if ($existingArgs -ne '') {
            $argumentList += $existingArgs -split '\s+'
        }
        foreach ($flag in @('/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART', '/CLOSEAPPLICATIONS')) {
            if ($argumentList -notcontains $flag) {
                $argumentList += $flag
            }
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
