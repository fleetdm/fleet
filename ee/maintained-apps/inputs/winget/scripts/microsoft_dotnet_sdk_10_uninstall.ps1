# Removes every .NET SDK 10 (x64) installed by Microsoft's standalone installer,
# across all feature bands and patch versions. SDKs installed by Visual Studio
# ("... (x64) from Visual Studio") are left for Visual Studio to manage. Shared
# components, like the .NET runtime, stay while another product depends on them.

$sdkMajor = '10'
$namePattern = "^Microsoft \.NET SDK $sdkMajor\.\d+\.\d+ \(x64\)$"

function Get-SdkBundles {
    $roots = @(
        'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
        'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall'
    )
    $found = @()
    foreach ($root in $roots) {
        foreach ($sub in (Get-ChildItem -Path $root -ErrorAction SilentlyContinue)) {
            $key = Get-ItemProperty -LiteralPath $sub.PSPath -ErrorAction SilentlyContinue
            if (-not $key -or $key.DisplayName -notmatch $namePattern) { continue }
            if ($key.Publisher -ne 'Microsoft Corporation') { continue }
            $found += $key
        }
    }
    return @($found)
}

function Get-BundleExe {
    param($Key)
    if ($Key.BundleCachePath) { return $Key.BundleCachePath }
    $u = if ($Key.QuietUninstallString) { $Key.QuietUninstallString } else { $Key.UninstallString }
    if (-not $u) { return $null }
    if ($u -match '^\s*"([^"]+)"') { return $Matches[1] }
    if ($u -match '(?i)^\s*(.+?\.exe)(\s|$)') { return $Matches[1] }
    return $null
}

$exitCode = 0

try {
    $bundles = @(Get-SdkBundles)
    if ($bundles.Count -eq 0) {
        Write-Host "No .NET SDK $sdkMajor (x64) installations found."
        Exit 0
    }

    foreach ($b in $bundles) {
        $exe = Get-BundleExe $b
        if (-not $exe -or -not (Test-Path -LiteralPath $exe)) {
            Write-Host "Cached installer missing for '$($b.DisplayName)': $exe"
            if ($exitCode -eq 0) { $exitCode = 1 }
            continue
        }

        Write-Host "Uninstalling '$($b.DisplayName)' via $exe"
        $p = Start-Process -FilePath $exe -ArgumentList "/uninstall /quiet /norestart" -PassThru -Wait -NoNewWindow
        Write-Host "  Exit code: $($p.ExitCode)"
        # 3010 = success, reboot required; 1641 = success, reboot initiated
        if (@(0, 3010, 1641) -notcontains $p.ExitCode -and $exitCode -eq 0) { $exitCode = $p.ExitCode }
    }

    $remaining = @(Get-SdkBundles)
    if ($remaining.Count -gt 0) {
        Write-Host "Still installed after uninstall:"
        $remaining | ForEach-Object { Write-Host "  - $($_.DisplayName)" }
        if ($exitCode -eq 0) { $exitCode = 1 }
    }
} catch {
    Write-Host "Error: $_"
    $exitCode = 1
}

Exit $exitCode
