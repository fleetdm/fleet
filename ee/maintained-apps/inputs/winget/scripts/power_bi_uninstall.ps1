# Power BI Desktop's EXE installer is a WiX "Burn" bundle. It registers TWO
# uninstall entries:
#   * "Microsoft PowerBI Desktop (x64)"  -> the bundle bootstrapper
#       (...\Package Cache\{afa18d15-...}\PBIDesktopSetup_x64.exe)
#   * "Microsoft Power BI Desktop (x64)" -> the MSI the bundle installed
#       (MsiExec.exe /X{c7d2053f-...})
#
# Removing the MSI directly orphans the bundle: its uninstall then no-ops
# (returns 0) and leaves the "Microsoft PowerBI Desktop (x64)" registration
# behind, which is what Fleet's osquery-based validator keeps detecting.
#
# Correct approach: uninstall via the BUNDLE first. It removes both the MSI and
# its own registration. Burn relaunches a cached copy of itself and spawns
# msiexec asynchronously, so wait for those to finish.

$ExpectedExitCodes = @(0, 1605, 1641, 3010)
$exitCode = 0

# Power BI's default install folder; used as a safety check before deleting any
# stale registration that survives a successful uninstall.
$installDirs = @(
    (Join-Path $env:ProgramFiles 'Microsoft Power BI Desktop'),
    (Join-Path ${env:ProgramFiles(x86)} 'Microsoft Power BI Desktop')
)

function Wait-ForProcessExit {
    param([string[]]$Names, [int]$TimeoutSeconds = 240)
    $elapsed = 0
    while ($elapsed -lt $TimeoutSeconds) {
        $running = $Names | Where-Object { Get-Process -Name $_ -ErrorAction SilentlyContinue }
        if (-not $running) { break }
        Start-Sleep -Seconds 3
        $elapsed += 3
    }
}

function Get-PowerBIEntries {
    param([string[]]$Roots)
    $list = @()
    foreach ($root in $Roots) {
        foreach ($sub in (Get-ChildItem -Path $root -ErrorAction SilentlyContinue)) {
            $key = Get-ItemProperty $sub.PSPath -ErrorAction SilentlyContinue
            if (-not $key.DisplayName) { continue }
            if (-not (($key.DisplayName -replace '\s', '').ToLower().Contains("powerbidesktop"))) { continue }
            $list += [PSCustomObject]@{
                DisplayName = $key.DisplayName
                KeyPath     = $sub.PSPath
                KeyName     = $sub.PSChildName
                Command     = if ($key.QuietUninstallString) { $key.QuietUninstallString } else { $key.UninstallString }
            }
        }
    }
    return $list
}

function Get-ExePath {
    param([string]$Command)
    if ($Command -match '"([^"]+\.exe)"') { return $Matches[1] }
    if ($Command -match '(?i)([A-Z]:\\[^"]+?\.exe)') { return $Matches[1] }
    return $null
}

try {

    # Uninstall roots across all hives (matches osquery's "programs" table).
    $roots = [System.Collections.Generic.List[string]]::new()
    $roots.Add('HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall')
    $roots.Add('HKLM:\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall')
    foreach ($hive in (Get-ChildItem 'Registry::HKEY_USERS' -ErrorAction SilentlyContinue)) {
        if ($hive.Name -match '_Classes$') { continue }
        $roots.Add("Registry::$($hive.Name)\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall")
        $roots.Add("Registry::$($hive.Name)\SOFTWARE\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall")
    }

    $entries = Get-PowerBIEntries -Roots $roots
    if ($entries.Count -eq 0) {
        Write-Host "No Power BI Desktop entries found (already removed)."
        Exit 0
    }

    foreach ($proc in @("PBIDesktop", "msmdsrv", "Microsoft.Mashup.Container")) {
        Stop-Process -Name $proc -Force -ErrorAction SilentlyContinue
    }

    # --- Phase 1: uninstall via the Burn bundle bootstrapper(s) FIRST. ---
    foreach ($e in ($entries | Where-Object { $_.Command -match "(?i)PBIDesktopSetup.*\.exe" })) {
        $exe = Get-ExePath $e.Command
        if (-not $exe) { Write-Host "Could not parse bundle exe from: $($e.Command)"; continue }
        if (-not (Test-Path -LiteralPath $exe)) { Write-Host "Bundle exe missing: $exe"; continue }

        Write-Host "Uninstalling bundle: '$($e.DisplayName)'"
        Write-Host "  Command: $exe"
        Write-Host "  Args: /uninstall /quiet /norestart"
        $p = Start-Process -FilePath $exe -ArgumentList "/uninstall /quiet /norestart" -PassThru -Wait
        Write-Host "  Exit code: $($p.ExitCode)"
        if (($ExpectedExitCodes -notcontains $p.ExitCode) -and ($exitCode -eq 0)) { $exitCode = $p.ExitCode }

        Wait-ForProcessExit -Names @("PBIDesktopSetup_x64", "PBIDesktopSetup", "msiexec") -TimeoutSeconds 240
    }

    # --- Phase 2: remove any MSI entries the bundle didn't clean up. ---
    foreach ($e in (Get-PowerBIEntries -Roots $roots)) {
        $msiCode = $null
        if ($e.Command -match "(?i)MsiExec\.exe\s+/[IX]\s*(\{[A-F0-9-]+\})") { $msiCode = $Matches[1] }
        elseif ($e.KeyName -match "(?i)^\{[A-F0-9-]+\}$") { $msiCode = $e.KeyName }
        if (-not $msiCode) { continue }

        Write-Host "Removing leftover MSI: '$($e.DisplayName)' ($msiCode)"
        $p = Start-Process -FilePath "MsiExec.exe" -ArgumentList "/X $msiCode /qn /norestart" -PassThru -Wait
        Write-Host "  Exit code: $($p.ExitCode)"
        Wait-ForProcessExit -Names @("msiexec") -TimeoutSeconds 180
        if (($ExpectedExitCodes -notcontains $p.ExitCode) -and ($exitCode -eq 0)) { $exitCode = $p.ExitCode }
    }

    # --- Phase 3: safety net. If the product files are gone but a stale ARP
    # registration lingers, remove the orphaned key so detection clears. Gated
    # on the install folder being absent so we never hide a real install. ---
    $productGone = -not ($installDirs | Where-Object { $_ -and (Test-Path -LiteralPath $_) })
    foreach ($e in (Get-PowerBIEntries -Roots $roots)) {
        $isMachineKey = $e.KeyPath -like 'Microsoft.PowerShell.Core\Registry::HKEY_LOCAL_MACHINE\*'
        if ($productGone -and $isMachineKey) {
            Write-Host "Removing orphaned registration: '$($e.DisplayName)' ($($e.KeyPath))"
            Remove-Item -Path $e.KeyPath -Recurse -Force -ErrorAction SilentlyContinue
        } else {
            Write-Host "WARNING: entry still present and product files remain: '$($e.DisplayName)'"
            if ($exitCode -eq 0) { $exitCode = 1 }
        }
    }

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

if ($ExpectedExitCodes -contains $exitCode) { Exit 0 } else { Exit $exitCode }
