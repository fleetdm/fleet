# Uninstall Jabra Direct (WiX burn bundle).
#
# Jabra Direct ships as a WiX burn bundle that also drops one or more MSI
# *components* in the registry. We must uninstall the bundle's own ARP entry
# (whose UninstallString is the cached bundle setup.exe, run with
# /uninstall /quiet /norestart), NOT an MSI component. Matching the component
# and shelling MsiExec.exe /I{GUID} would run an install/repair, not an
# uninstall (the original failure: exit 1).
#
# It is recommended to use an exact software name here if possible to avoid
# uninstalling unintended software.
$softwareNameLike = "*Jabra Direct*"

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

# Collect every matching entry, then prefer the burn bundle (a setup.exe-style
# UninstallString) over any MSI component (MsiExec.exe /I{GUID} or /X{GUID}).
# NOTE: do not name this $matches — that is the automatic variable populated by
# the -match operator and would be clobbered by the parsing below.
[array]$matchedKeys = $uninstallKeys | Where-Object { $_.DisplayName -like $softwareNameLike }

$bundleKeys = @($matchedKeys | Where-Object {
    $u = $_.QuietUninstallString
    if ([string]::IsNullOrWhiteSpace($u)) { $u = $_.UninstallString }
    $u -match '(?i)\.exe' -and $u -notmatch '(?i)^\s*MsiExec\.exe'
})

if ($bundleKeys.Count -gt 0) {
    $orderedKeys = @($bundleKeys) + @($matchedKeys | Where-Object { $bundleKeys -notcontains $_ })
} else {
    $orderedKeys = $matchedKeys
}

$foundUninstaller = $false
foreach ($key in $orderedKeys) {
    $foundUninstaller = $true

    # Prefer QuietUninstallString when present.
    $uninstallCommand = if ($key.QuietUninstallString) {
        $key.QuietUninstallString
    } else {
        $key.UninstallString
    }

    if ([string]::IsNullOrWhiteSpace($uninstallCommand)) {
        continue
    }

    # Parse the UninstallString defensively. It can be:
    #   "C:\ProgramData\Package Cache\{GUID}\setup.exe" /uninstall   (quoted)
    #   C:\path with spaces\setup.exe /uninstall                     (unquoted, may contain spaces)
    #   MsiExec.exe /I{GUID}  or  MsiExec.exe /X{GUID}               (bare token)
    $exe = $null
    $existingArgs = ""
    if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') {
        $exe = $matches[1]
        $existingArgs = $matches[2].Trim()
    } elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') {
        $exe = $matches[1]
        $existingArgs = $matches[2].Trim()
    } elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') {
        $exe = $matches[1]
        $existingArgs = $matches[2].Trim()
    } else {
        Throw "Could not parse uninstall command: $uninstallCommand"
    }

    if ($exe -match '(?i)MsiExec\.exe$') {
        # MSI component fallback. The bundle's components register with /I{GUID}
        # (install/repair). Rewrite to /X{GUID} so we actually uninstall.
        $existingArgs = $existingArgs -replace '(?i)/I({)', '/X$1'
        $uninstallArgs = $existingArgs
        if ($uninstallArgs -notmatch '(?i)/quiet|/qn') {
            $uninstallArgs = "$uninstallArgs /quiet".Trim()
        }
        if ($uninstallArgs -notmatch '(?i)/norestart') {
            $uninstallArgs = "$uninstallArgs /norestart".Trim()
        }
    } else {
        # Burn bundle setup.exe. Uninstall silently.
        $uninstallArgs = $existingArgs
        if ($uninstallArgs -notmatch '(?i)/uninstall') {
            $uninstallArgs = "$uninstallArgs /uninstall".Trim()
        }
        if ($uninstallArgs -notmatch '(?i)/quiet|/silent') {
            $uninstallArgs = "$uninstallArgs /quiet".Trim()
        }
        if ($uninstallArgs -notmatch '(?i)/norestart') {
            $uninstallArgs = "$uninstallArgs /norestart".Trim()
        }
    }

    Write-Host "Uninstall command: $exe"
    Write-Host "Uninstall args: $uninstallArgs"

    $processOptions = @{
        FilePath = $exe
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

if (-not $foundUninstaller) {
    Write-Host "Uninstall entry not found for $softwareNameLike"
    Exit 0
}

Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
