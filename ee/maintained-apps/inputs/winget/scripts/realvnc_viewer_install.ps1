try {
    # Extract the zip into a fresh temp dir.
    $extractDir = Join-Path $env:TEMP "vncviewer-extract-$(New-Guid)"
    New-Item -ItemType Directory -Path $extractDir -Force | Out-Null
    Write-Host "Extracting zip to: $extractDir"
    Expand-Archive -LiteralPath $zipPath -DestinationPath $extractDir -Force

    # Pick the MSI for the host architecture. On x64 Windows we want the
    # 64-bit MSI; fall back to 32-bit otherwise. Match the filename pattern
    # from the manifest ("...64bit.msi" / "...32bit.msi").
    if ([Environment]::Is64BitOperatingSystem) {
        $msi = Get-ChildItem -Path $extractDir -Filter "*64bit*.msi" -Recurse | Select-Object -First 1
    } else {
        $msi = Get-ChildItem -Path $extractDir -Filter "*32bit*.msi" -Recurse | Select-Object -First 1
    }

    if (-not $msi) {
        Throw "Could not find a VNC Viewer MSI under $extractDir for this architecture."
    }

    Write-Host "Install command: MsiExec.exe"
    Write-Host "Install args: /i `"$($msi.FullName)`" /quiet /norestart"

    $processOptions = @{
        FilePath     = "MsiExec.exe"
        ArgumentList = "/i", $msi.FullName, "/quiet", "/norestart"
        PassThru     = $true
        Wait         = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode
    Write-Host "Install exit code: $exitCode"

    # msiexec can return before the install fully completes; wait it out so
    # detection sees a settled state and the extract dir can be cleaned up.
    $elapsed = 0
    while ((Get-Process -Name "msiexec" -ErrorAction SilentlyContinue) -and ($elapsed -lt 120)) {
        Start-Sleep -Seconds 3
        $elapsed += 3
    }

    # Clean up extracted MSIs.
    Remove-Item -LiteralPath $extractDir -Recurse -Force -ErrorAction SilentlyContinue

    if ($ExpectedExitCodes -contains $exitCode) { Exit 0 }
    Exit $exitCode

} catch {
    Write-Host "Error: $_"
    Exit 1
}
