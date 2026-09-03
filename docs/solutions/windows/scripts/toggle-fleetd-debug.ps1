# Use this script to toggle debug mode for fleetd (Orbit) troubleshooting

$serviceName = "Fleet osquery"
$regPath = "HKLM:\SYSTEM\CurrentControlSet\Services\$serviceName"

if (-not (Test-Path $regPath)) {
    Write-Error "$serviceName service not found."
    exit 1
}

$imagePath = (Get-ItemProperty -Path $regPath -Name ImagePath).ImagePath

if (-not $imagePath) {
    Write-Error "ImagePath not found."
    exit 1
}

$imagePath = $imagePath.Trim()

if ($imagePath -match '\s--') {
    # Legacy install: fleetd's settings are command-line flags on the service ImagePath.
    $hasDebug = $imagePath -match '(^|\s)--debug(\s|$)'

    if ($hasDebug) {
        Write-Host "--debug is present: removing it."
        $imagePath = ($imagePath -replace '\s*--debug\b\s*').Trim()
    } else {
        Write-Host "--debug is missing: adding it."
        $imagePath = "$imagePath --debug"
    }

    Set-ItemProperty -Path $regPath -Name ImagePath -Value $imagePath -Type ExpandString
} else {
    # Current install: fleetd's settings are per-service environment variables in the
    # service's Environment (REG_MULTI_SZ) value, applied by Windows when the service starts.
    $environment = @((Get-ItemProperty -Path $regPath -Name Environment -ErrorAction SilentlyContinue).Environment | Where-Object { $_ })
    $hasDebug = @($environment | Where-Object { $_ -match '^ORBIT_DEBUG=' }).Count -gt 0

    if ($hasDebug) {
        Write-Host "ORBIT_DEBUG is present: removing it."
        $environment = @($environment | Where-Object { $_ -notmatch '^ORBIT_DEBUG=' })
    } else {
        Write-Host "ORBIT_DEBUG is missing: adding it."
        $environment += 'ORBIT_DEBUG=true'
    }

    Set-ItemProperty -Path $regPath -Name Environment -Value ([string[]]$environment) -Type MultiString
}

try {
    Restart-Service -Name $serviceName -Force -ErrorAction Stop
    Write-Host "$serviceName service restarted."
} catch {
    Write-Warning "$serviceName service restart failed: $_."
}

Write-Host "`nLogs are located at:"
Write-Host "C:\Windows\system32\config\systemprofile\AppData\Local\FleetDM\Orbit\Logs\orbit-osquery.log"
