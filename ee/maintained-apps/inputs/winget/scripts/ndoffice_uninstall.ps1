# Locate the ndOffice uninstaller from the registry and execute it silently.
# DisplayName is "NetDocuments ndOffice" (MSI ProductName), Publisher "NetDocuments".

$displayName = "NetDocuments ndOffice"
$publisher = "NetDocuments"

$paths = @(
    'HKLM:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall',
    'HKCU:\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall'
)

$uninstall = $null
foreach ($p in $paths) {
    $items = Get-ItemProperty "$p\*" -ErrorAction SilentlyContinue | Where-Object {
        $dn = $_.DisplayName
        if (-not $dn) { return $false }
        if ($publisher -ne "" -and $_.Publisher -ne $publisher) { return $false }
        ($dn -eq $displayName -or $dn -like "$displayName*")
    }
    if ($items) { $uninstall = $items | Select-Object -First 1; break }
}

if (-not $uninstall -or -not $uninstall.UninstallString) {
    Write-Host "Uninstall entry not found"
    Exit 0
}

$uninstallCommand = $uninstall.UninstallString

if ($uninstallCommand -match "MsiExec\.exe\s+/[IX]\s*(\{[A-F0-9-]+\})") {
    $productCode = $Matches[1]
    $uninstallArgs = "/X $productCode /qn /norestart"
    $uninstallCommand = "MsiExec.exe"
} else {
    Write-Host "Error: Unable to parse uninstall command: $uninstallCommand"
    Exit 1
}

Write-Host "Uninstall command: $uninstallCommand"
Write-Host "Uninstall args: $uninstallArgs"

try {
    $process = Start-Process -FilePath $uninstallCommand -ArgumentList $uninstallArgs -NoNewWindow -PassThru -Wait
    $exitCode = $process.ExitCode
    Write-Host "Uninstall exit code: $exitCode"

    $timeout = 60
    $elapsed = 0
    while ((Get-Process -Name "msiexec" -ErrorAction SilentlyContinue) -and ($elapsed -lt $timeout)) {
        Start-Sleep -Seconds 2
        $elapsed += 2
        Write-Host "Waiting for MsiExec to complete... ($elapsed seconds)"
    }

    # 3010 = success, reboot required; 1641 = success, reboot initiated.
    if ($exitCode -eq 3010 -or $exitCode -eq 1641) {
        Exit 0
    }

    Exit $exitCode
} catch {
    Write-Host "Error running uninstaller: $_"
    Exit 1
}
