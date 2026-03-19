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

$hasDebug = $imagePath -match '(^|\s)--debug(\s|$)'

if ($hasDebug) {
    Write-Host "--debug is present: removing it."
    $imagePath = ($imagePath -replace '\s*--debug\b\s*').Trim()
} else {
    Write-Host "--debug is missing: adding it."
    $imagePath = "$imagePath --debug"
}

Set-ItemProperty -Path $regPath -Name ImagePath -Value $imagePath -Type ExpandString

try {
    Restart-Service -Name $serviceName -Force -ErrorAction Stop
    Write-Host "$serviceName service restarted."
} catch {
    Write-Warning "$serviceName service restart failed: $_."
}

Write-Host "`nLogs are located at:"
Write-Host "C:\Windows\system32\config\systemprofile\AppData\Local\FleetDM\Orbit\Logs\orbit-osquery.log"

