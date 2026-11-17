# Learn more about .exe install scripts:
# http://fleetdm.com/learn-more-about/exe-install-scripts
#
# DataGrip ships as a Nullsoft (NSIS) installer; /S runs it silently.

$exeFilePath = "${env:INSTALLER_PATH}"

try {

# In silent mode the installer refuses a non-empty install directory with a
# message box. Fleet runs it as SYSTEM, where nobody can dismiss that box, so a
# reinstall of the same version would hang. Empty the directory first (it's
# "DataGrip <ProductVersion>"), stopping DataGrip if it's running from there.
# Unsaved changes are lost.
$version = (Get-Item $exeFilePath).VersionInfo.ProductVersion
$programFiles = if (${env:ProgramW6432}) { ${env:ProgramW6432} } else { ${env:ProgramFiles} }
$installDir = Join-Path $programFiles "JetBrains\DataGrip $version"

if ($version -and (Test-Path -LiteralPath $installDir)) {
    $running = Get-Process -ErrorAction SilentlyContinue |
        Where-Object { $_.Path -like "$installDir\*" }
    foreach ($proc in $running) {
        Write-Host "Stopping $($proc.Name) (PID $($proc.Id)) before installing."
        Stop-Process -Id $proc.Id -Force -ErrorAction SilentlyContinue
    }
    $running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue

    # A stopped DataGrip's exe and DLLs can stay locked for a few seconds.
    Write-Host "Removing the existing $installDir before reinstalling."
    for ($attempt = 0; $attempt -lt 30; $attempt++) {
        Remove-Item -LiteralPath $installDir -Recurse -Force -ErrorAction SilentlyContinue
        $left = @(Get-ChildItem -LiteralPath $installDir -Recurse -File -Force -ErrorAction SilentlyContinue)
        if ($left.Count -eq 0) { break }
        Start-Sleep -Seconds 2
    }
    if ($left.Count -gt 0) {
        Write-Host "Error: $($left.Count) file(s) in $installDir couldn't be removed (first: $($left[0].FullName)). The installer would hang on a non-empty directory."
        Exit 1
    }
}

# On Windows on Arm the installer refuses this x64 build with a message box
# that hangs under SYSTEM. /IGNORE_ARCH skips that check; the app runs under
# x64 emulation.
$processOptions = @{
  FilePath = "$exeFilePath"
  ArgumentList = "/S /IGNORE_ARCH"
  PassThru = $true
  Wait = $true
}

$process = Start-Process @processOptions
$exitCode = $process.ExitCode

Write-Host "Install exit code: $exitCode"

# Each version installs to its own directory, and a silent install leaves older
# versions in place. Uninstall them (settings are kept), stopping any that are
# running. Unsaved changes are lost.
if ($exitCode -eq 0) {
    try {
        $entries = @(Get-ChildItem -Path @(
                'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
                'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
            ) -ErrorAction SilentlyContinue |
            ForEach-Object { Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue } |
            Where-Object { $_.DisplayName -match '^DataGrip \d' -and $_.Publisher -like '*JetBrains*' -and $_.InstallLocation })
        $current = $entries | Where-Object { $_.InstallLocation.TrimEnd('\') -eq $installDir } | Select-Object -First 1
        $currentBuild = $null
        if ($current) { [void][version]::TryParse($current.DisplayVersion, [ref]$currentBuild) }

        foreach ($entry in $entries) {
            $build = $null
            if (-not $currentBuild -or -not [version]::TryParse($entry.DisplayVersion, [ref]$build) -or $build -ge $currentBuild) { continue }
            $oldDir = $entry.InstallLocation.TrimEnd('\')
            # The uninstaller shows a message box, which hangs under SYSTEM, if its
            # launcher is missing or the IDE is running.
            if (-not (Test-Path -LiteralPath "$oldDir\bin\Uninstall.exe") -or -not (Test-Path -LiteralPath "$oldDir\bin\datagrip64.exe")) {
                Write-Host "Warning: can't remove $($entry.DisplayName) from $oldDir (uninstaller or launcher missing)."
                continue
            }
            $running = @(Get-Process -ErrorAction SilentlyContinue | Where-Object { $_.Path -like "$oldDir\*" })
            $running | Stop-Process -Force -ErrorAction SilentlyContinue
            $running | Wait-Process -Timeout 30 -ErrorAction SilentlyContinue

            # Run a copy so the uninstaller can delete itself; _?= keeps it in the
            # foreground and must be the last argument, unquoted.
            $tempDir = Join-Path ([IO.Path]::GetTempPath()) ([guid]::NewGuid())
            New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
            Copy-Item -LiteralPath "$oldDir\bin\Uninstall.exe" -Destination $tempDir
            Write-Host "Uninstalling $($entry.DisplayName) from $oldDir."
            $uninstall = Start-Process -FilePath "$tempDir\Uninstall.exe" -ArgumentList "/S /NO_UNINSTALL_FEEDBACK=true _?=$oldDir\bin" -PassThru
            $null = $uninstall.Handle
            if ($uninstall.WaitForExit(600000)) {
                Write-Host "Uninstall exit code: $($uninstall.ExitCode)"
            } else {
                Stop-Process -Id $uninstall.Id -Force -ErrorAction SilentlyContinue
                Write-Host "Warning: uninstalling $($entry.DisplayName) timed out after 10 minutes."
            }
            # The copy can stay locked for a moment after the uninstaller exits.
            for ($attempt = 0; $attempt -lt 10; $attempt++) {
                Remove-Item -LiteralPath $tempDir -Recurse -Force -ErrorAction SilentlyContinue
                if (-not (Test-Path -LiteralPath $tempDir)) { break }
                Start-Sleep -Seconds 1
            }
            if (Test-Path -LiteralPath $entry.PSPath) {
                Write-Host "Warning: $($entry.DisplayName) is still installed in $oldDir."
            }
        }
    } catch {
        Write-Host "Warning: couldn't remove older versions: $_"
    }
}

Exit $exitCode

} catch {
  Write-Host "Error: $_"
  Exit 1
}
