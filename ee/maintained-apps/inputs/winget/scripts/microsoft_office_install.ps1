# Microsoft Office (Click-to-Run) bootstrap installer.
# setup.exe is a small (~7 MB) Office Deployment Tool bootstrap. Rather than
# fetching the configuration XML from a Microsoft-hosted URL at install time,
# we write a pinned configuration to disk and point setup.exe at it with
# /configure. This makes the install behavior fully determined by this script
# and immune to changes in the remote config (e.g. Display Level flipping to
# Full, or the product set changing) and to that URL being unreachable.
# http://fleetdm.com/learn-more-about/exe-install-scripts

$exeFilePath = "${env:INSTALLER_PATH}"

# Office Deployment Tool configuration. Equivalent to the config Microsoft
# serves at https://aka.ms/fhlwingetconfig, pinned here for deterministic,
# offline-safe installs.
#   Display Level="None"  -> fully silent, no setup UI is shown.
#   AcceptEULA="TRUE"     -> license terms are accepted automatically, so the
#                            user is not prompted on first launch of an Office
#                            app (required for a hands-off fleet deployment).
#   RemoveMSI             -> removes any pre-existing MSI (volume/perpetual)
#                            Office before installing Click-to-Run.
# Visio/Project install only when a matching MSI product is already present
# (MSICondition), so a clean machine receives Microsoft 365 Apps only.
$configXml = @'
<Configuration>
  <Add>
    <Product ID="O365ProPlusRetail">
      <Language ID="MatchOS"/>
      <Language ID="MatchPreviousMSI"/>
      <ExcludeApp ID="Groove"/>
      <ExcludeApp ID="Lync"/>
    </Product>
    <Product ID="VisioProRetail" MSICondition="VisPro,VisProR">
      <Language ID="MatchOS"/>
      <Language ID="MatchPreviousMSI"/>
      <ExcludeApp ID="Groove"/>
      <ExcludeApp ID="Lync"/>
    </Product>
    <Product ID="ProjectProRetail" MSICondition="PrjPro,PrjProR">
      <Language ID="MatchOS"/>
      <Language ID="MatchPreviousMSI"/>
      <ExcludeApp ID="Groove"/>
      <ExcludeApp ID="Lync"/>
    </Product>
  </Add>
  <RemoveMSI/>
  <Display Level="None" AcceptEULA="TRUE"/>
</Configuration>
'@

# Write the configuration next to the installer. Use .NET WriteAllText so the
# file is UTF-8 without a BOM, which the Office Deployment Tool parses cleanly.
$configPath = Join-Path $env:TEMP "fleet-office-config.xml"
[System.IO.File]::WriteAllText($configPath, $configXml, (New-Object System.Text.UTF8Encoding $false))

$exitCode = 0

try {
    # setup.exe /configure runs synchronously. It blocks until Click-to-Run
    # has downloaded and applied all selected products, which can take
    # 15-60+ minutes depending on network speed and selected products.
    $processOptions = @{
        FilePath     = "$exeFilePath"
        ArgumentList = "/configure `"$configPath`""
        PassThru     = $true
        Wait         = $true
        NoNewWindow  = $true
    }

    Write-Host "Starting Microsoft Office setup: $exeFilePath /configure $configPath"
    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "setup.exe exit code: $exitCode"

    if ($exitCode -ne 0) {
        Exit $exitCode
    }

    # On some builds, the bootstrap returns before Click-to-Run has finished
    # writing the ARP uninstall entry. Poll briefly so that subsequent
    # osquery inventory picks the app up on the next run.
    $maxWaitSeconds = 600
    $elapsed = 0
    $registered = $null
    while ($elapsed -lt $maxWaitSeconds -and -not $registered) {
        foreach ($root in @(
                'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
                'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
            )) {
            $match = Get-ItemProperty "$root\*" -ErrorAction SilentlyContinue | Where-Object {
                $_.Publisher -eq 'Microsoft Corporation' -and
                $_.DisplayName -and
                ($_.DisplayName -like 'Microsoft 365*' -or $_.DisplayName -like 'Microsoft Office*') -and
                $_.UninstallString -like '*OfficeClickToRun.exe*'
            }
            if ($match) {
                $registered = $match | Select-Object -First 1
                break
            }
        }
        if (-not $registered) {
            Start-Sleep -Seconds 10
            $elapsed += 10
        }
    }

    if ($registered) {
        Write-Host "Detected installed product: $($registered.DisplayName) ($($registered.DisplayVersion))"
    } else {
        Write-Host "Warning: Microsoft Office uninstall entry not detected after $maxWaitSeconds seconds"
    }

} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

Exit $exitCode
