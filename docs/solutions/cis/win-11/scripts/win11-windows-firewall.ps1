#Requires -RunAsAdministrator
<#
.SYNOPSIS
    CIS Windows 11 Enterprise Benchmark v4.0.0 - Windows Firewall Remediation

.DESCRIPTION
    Configures Windows Firewall settings to meet CIS Windows 11 Enterprise Benchmark v4.0.0
    requirements using Set-NetFirewallProfile. This script is idempotent.

.NOTES
    CIS Benchmark Controls Covered:
    - 9.1.1  Windows Firewall: Domain: Firewall state              -> On
    - 9.1.2  Windows Firewall: Domain: Inbound connections         -> Block
    - 9.1.3  Windows Firewall: Domain: Outbound connections        -> Allow
    - 9.1.4  Windows Firewall: Domain: Settings: Display notification -> No
    - 9.2.1  Windows Firewall: Private: Firewall state             -> On
    - 9.2.2  Windows Firewall: Private: Inbound connections        -> Block
    - 9.2.3  Windows Firewall: Private: Outbound connections       -> Allow
    - 9.2.4  Windows Firewall: Private: Settings: Display notification -> No
    - 9.3.1  Windows Firewall: Public: Firewall state              -> On
    - 9.3.2  Windows Firewall: Public: Inbound connections         -> Block
    - 9.3.3  Windows Firewall: Public: Outbound connections        -> Allow
    - 9.3.4  Windows Firewall: Public: Settings: Display notification -> No
    - 9.x.5  Windows Firewall: Log dropped packets                -> Yes
    - 9.x.6  Windows Firewall: Log max file size                  -> 16384 KB

    Requires: Administrator privileges, Windows Firewall service running
#>

[CmdletBinding(SupportsShouldProcess)]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Write-Status {
    param([string]$Message, [string]$Level = 'INFO')
    $timestamp = Get-Date -Format 'yyyy-MM-dd HH:mm:ss'
    Write-Output "[$timestamp] [$Level] $Message"
}

Write-Status "Starting CIS Windows Firewall remediation"

# Verify the Windows Firewall service is running
$fwService = Get-Service -Name 'MpsSvc' -ErrorAction SilentlyContinue
if ($null -eq $fwService) {
    Write-Status "Windows Firewall service (MpsSvc) not found." 'ERROR'
    exit 1
}
if ($fwService.Status -ne 'Running') {
    Write-Status "Starting Windows Firewall service..."
    Start-Service -Name 'MpsSvc'
}

try {
    $profiles = @('Domain', 'Private', 'Public')

    foreach ($profile in $profiles) {
        Write-Status "Configuring $profile firewall profile..."

        $params = @{
            Name                  = $profile
            Enabled               = $true            # Firewall state: On
            DefaultInboundAction  = 'Block'          # Block inbound connections by default
            DefaultOutboundAction = 'Allow'          # Allow outbound connections by default
            NotifyOnListen        = $false           # Do not display notifications
            AllowUnicastResponseToMulticast = $true  # Allow unicast response (recommended)
            LogAllowed            = $false
            LogBlocked            = $true            # Log dropped packets (CIS 9.x.5)
            LogMaxSizeKilobytes   = 16384            # CIS 9.x.6: minimum 16384 KB
        }

        # Public profile: also disable local policy merge (CIS L2 hardening)
        if ($profile -eq 'Public') {
            $params['AllowLocalFirewallRules'] = $false
            $params['AllowLocalIPsecRules']    = $false
        }

        Set-NetFirewallProfile @params
        Write-Status "$profile profile configured."
    }

    # Verify settings were applied
    Write-Status "Verifying firewall configuration..."
    foreach ($profile in $profiles) {
        $fw = Get-NetFirewallProfile -Name $profile
        $status = if ($fw.Enabled) { 'ON' } else { 'OFF' }
        Write-Status "  $profile : State=$status, Inbound=$($fw.DefaultInboundAction), Outbound=$($fw.DefaultOutboundAction)"
    }

} catch {
    Write-Status "ERROR: $_" 'ERROR'
    exit 1
}

Write-Status "Windows Firewall remediation complete."
