# Please don't delete. This script is used in the guide here: https://fleetdm.com/guides/scripts

# Error if not run as admin
if (-not ([Security.Principal.WindowsPrincipal] [Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    Write-Error "This script must be run as an administrator."
    exit 1
}

$serviceName = "Fleet osquery"
$regPath = "HKLM:\SYSTEM\CurrentControlSet\Services\$serviceName"

$service = Get-CimInstance -ClassName Win32_Service -Filter "Name='$serviceName'"
if (-not $service) {
    Write-Error "Service '$serviceName' not found."
    exit 1
}

if ($service.PathName -match '\s--') {
    # Legacy install: fleetd's settings are command-line flags on the service ImagePath.
    # Replace any existing --enable-scripts flag with --enable-scripts="True"
    $modifiedPath = $service.PathName -replace '--enable-scripts(=".*?")?', '--enable-scripts="True"'
    sc.exe config "$serviceName" binPath= "$modifiedPath"
} else {
    # Current install: fleetd's settings are per-service environment variables in the
    # service's Environment (REG_MULTI_SZ) value, applied by Windows when the service starts.
    $environment = @((Get-ItemProperty -Path $regPath -Name Environment -ErrorAction SilentlyContinue).Environment | Where-Object { $_ })
    $environment = @($environment | Where-Object { $_ -notmatch '^ORBIT_ENABLE_SCRIPTS=' }) + 'ORBIT_ENABLE_SCRIPTS=True'
    Set-ItemProperty -Path $regPath -Name Environment -Value ([string[]]$environment) -Type MultiString
}

# Restart the service
Restart-Service -Name $serviceName
Write-Host "Scripts enabled and service restarted."
