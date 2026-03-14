# Security Baseline - Service Configuration
# Disables services that increase attack surface and have no CSP equivalent.
# Delivered via Fleet script execution; verified by osquery service policies.

$ErrorActionPreference = 'Stop'

$servicesToDisable = @(
    'RemoteRegistry',  # Remote Registry - prevents remote registry access
    'SSDPSRV',         # SSDP Discovery - UPnP device discovery
    'upnphost'         # UPnP Device Host - hosting UPnP devices
)

$results = @()

foreach ($svc in $servicesToDisable) {
    try {
        $service = Get-Service -Name $svc -ErrorAction SilentlyContinue
        if ($null -eq $service) {
            $results += "SKIP: Service '$svc' not found on this system."
            continue
        }

        if ($service.StartType -eq 'Disabled') {
            $results += "OK: Service '$svc' is already disabled."
            continue
        }

        # Stop if running
        if ($service.Status -eq 'Running') {
            Stop-Service -Name $svc -Force -ErrorAction Stop
        }

        # Disable
        Set-Service -Name $svc -StartupType Disabled -ErrorAction Stop
        $results += "CHANGED: Service '$svc' disabled successfully."
    }
    catch {
        $results += "ERROR: Failed to disable service '$svc': $_"
    }
}

# Output results
$results | ForEach-Object { Write-Output $_ }

# Exit with error if any failures
if ($results | Where-Object { $_ -like 'ERROR:*' }) {
    exit 1
}
exit 0
