# Uninstall PreForm (WiX burn bundle).
# Burn bundles register the bundle's own ARP entry (UninstallString = the cached
# setup.exe, run with /uninstall /quiet /norestart) alongside MSI *component*
# entries. Prefer the bundle (a .exe UninstallString) over any MsiExec component;
# shelling MsiExec.exe with /uninstall fails (exit 1619), and /I{GUID} would
# install/repair rather than uninstall.

$softwareNameLike = "*PreForm*"

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

[array]$matchedKeys = $uninstallKeys | Where-Object { $_.DisplayName -like $softwareNameLike }

# Prefer the burn bundle (.exe UninstallString, not MsiExec.exe) over MSI components.
$bundleKeys = @($matchedKeys | Where-Object {
    $u = $_.QuietUninstallString
    if ([string]::IsNullOrWhiteSpace($u)) { $u = $_.UninstallString }
    $u -match '(?i)\.exe' -and $u -notmatch '(?i)^\s*"?\s*MsiExec\.exe'
})
if ($bundleKeys.Count -gt 0) {
    $orderedKeys = @($bundleKeys) + @($matchedKeys | Where-Object { $bundleKeys -notcontains $_ })
} else {
    $orderedKeys = $matchedKeys
}

$foundUninstaller = $false
foreach ($key in $orderedKeys) {
    $foundUninstaller = $true
    $uninstallCommand = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }
    if ([string]::IsNullOrWhiteSpace($uninstallCommand)) { continue }

    # Parse defensively: quoted path, unquoted path-with-spaces, or bare token.
    $exe = $null; $existingArgs = ""
    if ($uninstallCommand -match '^\s*"([^"]+)"\s*(.*)$') { $exe = $matches[1]; $existingArgs = $matches[2].Trim() }
    elseif ($uninstallCommand -match '(?i)^\s*(.+?\.exe)\s*(.*)$') { $exe = $matches[1]; $existingArgs = $matches[2].Trim() }
    elseif ($uninstallCommand -match '^\s*(\S+)\s*(.*)$') { $exe = $matches[1]; $existingArgs = $matches[2].Trim() }
    else { Write-Host "Error: Could not parse uninstall command: $uninstallCommand"; Exit 1 }

    if ($exe -match '(?i)MsiExec\.exe$') {
        # MSI component fallback: rewrite /I{GUID} -> /X{GUID} so we uninstall.
        $uninstallArgs = $existingArgs -replace '(?i)/I({)', '/X$1'
        if ($uninstallArgs -notmatch '(?i)/quiet|/qn') { $uninstallArgs = "$uninstallArgs /quiet".Trim() }
        if ($uninstallArgs -notmatch '(?i)/norestart') { $uninstallArgs = "$uninstallArgs /norestart".Trim() }
    } else {
        # Burn bundle setup.exe: uninstall silently.
        $uninstallArgs = $existingArgs
        if ($uninstallArgs -notmatch '(?i)/uninstall') { $uninstallArgs = "$uninstallArgs /uninstall".Trim() }
        if ($uninstallArgs -notmatch '(?i)/quiet|/silent') { $uninstallArgs = "$uninstallArgs /quiet".Trim() }
        if ($uninstallArgs -notmatch '(?i)/norestart') { $uninstallArgs = "$uninstallArgs /norestart".Trim() }
    }

    Write-Host "Uninstall command: $exe"
    Write-Host "Uninstall args: $uninstallArgs"

    $processOptions = @{ FilePath = $exe; PassThru = $true; Wait = $true }
    if ($uninstallArgs -ne '') { $processOptions.ArgumentList = $uninstallArgs }
    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"
    break
}

if (-not $foundUninstaller) { Write-Host "Uninstall entry not found for $softwareNameLike"; Exit 0 }
Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
