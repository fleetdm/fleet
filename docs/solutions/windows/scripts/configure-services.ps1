# configure-services.ps1
# CIS Windows 11 Enterprise Benchmark v4.0.0 - Section 5 Service Configuration
# Disables services per CIS requirements

$ErrorActionPreference = 'SilentlyContinue'

$servicesToDisable = @(
    'BTAGService',        # 5.1  - Bluetooth Audio Gateway Service (L2)
    'bthserv',            # 5.2  - Bluetooth Support Service (L2)
    'Browser',            # 5.3  - Computer Browser (L1)
    'MapsBroker',         # 5.4  - Downloaded Maps Manager (L2)
    'GameInputSvc',       # 5.5  - GameInput Service (L2)
    'lfsvc',              # 5.6  - Geolocation Service (L2)
    'IISADMIN',           # 5.7  - IIS Admin Service (L1)
    'irmon',              # 5.8  - Infrared monitor service (L1)
    'lltdsvc',            # 5.9  - Link-Layer Topology Discovery Mapper (L2)
    'LxssManager',        # 5.10 - LxssManager / WSL (L1)
    'FTPSVC',             # 5.11 - Microsoft FTP Service (L1)
    'MSiSCSI',            # 5.12 - Microsoft iSCSI Initiator Service (L2)
    'sshd',               # 5.13 - OpenSSH SSH Server (L1)
    'wercplsupport',      # 5.15 - Problem Reports Control Panel Support (L2)
    'RasAuto',            # 5.16 - Remote Access Auto Connection Manager (L2)
    'SessionEnv',         # 5.17 - Remote Desktop Configuration (L2)
    'TermService',        # 5.18 - Remote Desktop Services (L2)
    'UmRdpService',       # 5.19 - Remote Desktop Services UserMode Port Redirector (L2)
    'RpcLocator',         # 5.20 - Remote Procedure Call (RPC) Locator (L1)
    'RemoteRegistry',     # 5.21 - Remote Registry (L2)
    'RemoteAccess',       # 5.22 - Routing and Remote Access (L1)
    'LanmanServer',       # 5.23 - Server (L2)
    'simptcp',            # 5.24 - Simple TCP/IP Services (L1)
    'SNMP',               # 5.25 - SNMP Service (L2)
    'sacsvr',             # 5.26 - Special Administration Console Helper (L1)
    'SSDPSRV',            # 5.27 - SSDP Discovery (L1)
    'upnphost',           # 5.28 - UPnP Device Host (L1)
    'WMSvc',              # 5.29 - Web Management Service (L1)
    'WerSvc',             # 5.30 - Windows Error Reporting Service (L2)
    'Wecsvc',             # 5.31 - Windows Event Collector (L2)
    'WMPNetworkSvc',      # 5.32 - Windows Media Player Network Sharing Service (L1)
    'icssvc',             # 5.33 - Windows Mobile Hotspot Service (L1)
    'WpnService',         # 5.34 - Windows Push Notifications System Service (L2)
    'PushToInstall',      # 5.35 - Windows PushToInstall Service (L2)
    'WinRM',              # 5.36 - Windows Remote Management (L2)
    'WinHttpAutoProxySvc',# 5.37 - WinHTTP Web Proxy Auto-Discovery Service (L2)
    'W3SVC',              # 5.38 - World Wide Web Publishing Service (L1)
    'XboxGipSvc',         # 5.39 - Xbox Accessory Management Service (L1)
    'XblAuthManager',     # 5.40 - Xbox Live Auth Manager (L1)
    'XblGameSave',        # 5.41 - Xbox Live Game Save (L1)
    'XboxNetApiSvc'       # 5.42 - Xbox Live Networking Service (L1)
)

foreach ($svc in $servicesToDisable) {
    $service = Get-Service -Name $svc -ErrorAction SilentlyContinue
    if ($service) {
        if ($service.Status -eq 'Running') {
            Stop-Service -Name $svc -Force -ErrorAction SilentlyContinue
        }
        Set-Service -Name $svc -StartupType Disabled -ErrorAction SilentlyContinue
        Write-Output "Disabled service: $svc"
    } else {
        Write-Output "Service not found (OK): $svc"
    }
}

Write-Output "Service configuration complete."
