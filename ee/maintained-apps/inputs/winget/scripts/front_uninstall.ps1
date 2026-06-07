# Locates Front's uninstaller from the registry and runs it silently.

# Match the registry DisplayName (osquery programs.name).
$softwareNameLike = "Front"

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

        # Build argument list, ensuring /S (silent).
        $argumentList = @()
        if ($existingArgs -ne '') {
            $argumentList += $existingArgs -split '\s+'
        }
        if ($argumentList -notcontains '/S') {
            $argumentList += '/S'
        }

        # Front is an electron-builder NSIS app. By default the NSIS uninstaller
        # copies itself to %TEMP% and re-launches from there, so the process we
        # start returns IMMEDIATELY while the real removal happens in a detached
        # child (Un_A.exe). With Wait=$true we'd still return before the app is
        # gone, leaving the entry behind (the validator re-found 'Front' after
        # uninstall). The NSIS `_?=<INSTDIR>` parameter disables that self-copy so
        # the uninstaller runs in place and our Wait actually blocks until removal
        # completes. It must be the LAST argument and an absolute path with no
        # trailing backslash, equal to the uninstaller's own directory.
        $instDir = (Split-Path -Path $exePath -Parent).TrimEnd('\')
        $argumentList += "_?=$instDir"

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
