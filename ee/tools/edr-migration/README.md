# Grubhub EDR Migration Toolkit — SentinelOne to CrowdStrike Falcon

This toolkit automates the migration of endpoint detection and response (EDR) software from **SentinelOne** to **CrowdStrike Falcon** across macOS and Windows endpoints. It is designed for enterprise deployment via Fleet, SCCM, Intune, or other MDM/management platforms.

Both platform scripts provide a branded **Grubhub IT** user experience with progress indicators, error handling, and logging.

## Directory Structure

```
ee/tools/edr-migration/
├── README.md                                  # This file
├── macos/
│   └── migrate_edr.sh                         # macOS migration script (SwiftDialog UI)
└── windows/
    ├── Invoke-AppDeployToolkit.ps1             # Windows migration script (PSADT v4.1)
    ├── AppDeployToolkitConfig.xml              # PSADT branding/configuration
    └── Files/                                  # (create this) Place installers here
        └── CrowdStrike/
            └── WindowsSensor.exe               # CrowdStrike Falcon installer
```

## Prerequisites

### macOS

| Requirement | Details |
|---|---|
| **macOS version** | 12.0 (Monterey) or later |
| **SwiftDialog** | v2.x+ installed at `/usr/local/bin/dialog`. Install via MDM before running the migration. Download from [swiftDialog releases](https://github.com/swiftDialog/swiftDialog/releases). |
| **Root access** | The script must run as root (standard for MDM-deployed scripts). |
| **CrowdStrike PKG** | The CrowdStrike Falcon sensor `.pkg` installer, staged on the endpoint or a network-accessible path. |
| **SentinelOne passphrase** | The management passphrase for SentinelOne (if anti-tamper is enabled). Obtain from the SentinelOne management console. |
| **CrowdStrike CID** | Your CrowdStrike Customer ID. Obtain from the Falcon console under **Host setup and management > Sensor downloads**. |

### Windows

| Requirement | Details |
|---|---|
| **Windows version** | Windows 10 1809+ or Windows 11 |
| **PSADT v4.1** | PowerShell App Deployment Toolkit v4.1 framework files in the `AppDeployToolkit\` subdirectory. Download from [psappdeploytoolkit.com](https://psappdeploytoolkit.com). |
| **Administrator rights** | The script must run with administrative privileges. |
| **CrowdStrike installer** | `WindowsSensor.exe` (or `.msi`) placed in `Files\CrowdStrike\`. |
| **SentinelOne passphrase** | The management passphrase/token for SentinelOne uninstall. |
| **CrowdStrike CID** | Your CrowdStrike Customer ID. |
| **PowerShell** | PowerShell 5.1 or later (built into Windows 10+). |

## Configuration

### macOS — `migrate_edr.sh`

Edit the configuration variables at the top of the script:

```bash
# Path to the CrowdStrike Falcon sensor installer PKG
CROWDSTRIKE_PKG_PATH="/path/to/CrowdStrikeFalcon.pkg"

# CrowdStrike Customer ID (CID)
CROWDSTRIKE_CID="YOUR-CID-HERE-WITH-CHECKSUM"

# SentinelOne management passphrase (if anti-tamper is enabled)
SENTINELONE_PASSPHRASE="your-passphrase-here"

# Support contact info
SUPPORT_EMAIL="it-security@grubhub.com"
SUPPORT_URL="https://grubhub.service-now.com"

# SwiftDialog icon (SF Symbol name or path to image)
DIALOG_ICON="shield.checkerboard"

# Optional banner image for dialogs
DIALOG_BANNER_IMAGE="/path/to/grubhub_banner.png"
```

### Windows — `Invoke-AppDeployToolkit.ps1`

Edit the configuration variables at the top of the script:

```powershell
$CrowdStrikeInstallerPath = ".\Files\CrowdStrike\WindowsSensor.exe"
$CrowdStrikeCID           = "YOUR-CID-HERE-WITH-CHECKSUM"
$SentinelOnePassphrase    = "your-passphrase-here"
$SupportEmail             = "it-security@grubhub.com"
$SupportURL               = "https://grubhub.service-now.com"
```

Additionally, update `AppDeployToolkitConfig.xml` if you want to customize branding elements (banner image, icon, colors).

## Deployment

### macOS via Fleet

1. Stage the CrowdStrike Falcon `.pkg` on each endpoint (via Fleet file distribution, a CDN, or a local cache).
2. Configure the variables in `migrate_edr.sh`.
3. Deploy the script via **Fleet > Scripts** or as a custom MDM profile:

```bash
# Upload and run via Fleet
fleetctl apply -f migrate_edr_policy.yml
```

Or run directly on a test machine:

```bash
sudo /bin/bash migrate_edr.sh
```

### macOS via other MDMs (Jamf, Mosyle, Kandji)

1. Upload `migrate_edr.sh` as a script in your MDM.
2. Set the script to run as root.
3. Scope it to the target devices/groups.
4. Ensure SwiftDialog and the CrowdStrike PKG are pre-staged.

### Windows via Fleet

1. Place the CrowdStrike installer in `Files\CrowdStrike\`.
2. Configure the variables in `Invoke-AppDeployToolkit.ps1`.
3. Package the entire `windows/` directory.
4. Deploy via **Fleet > Scripts** as a PowerShell script:

```powershell
powershell.exe -ExecutionPolicy Bypass -File "Invoke-AppDeployToolkit.ps1" -DeploymentType Install -DeployMode Interactive
```

### Windows via SCCM/MECM

1. Create an Application in SCCM.
2. Set the installation program to:
   ```
   powershell.exe -ExecutionPolicy Bypass -File "Invoke-AppDeployToolkit.ps1" -DeploymentType Install -DeployMode Interactive
   ```
3. Set the uninstall program to:
   ```
   powershell.exe -ExecutionPolicy Bypass -File "Invoke-AppDeployToolkit.ps1" -DeploymentType Uninstall -DeployMode Silent
   ```
4. Detection method: Check for `CSFalconService` running, or check registry for CrowdStrike Falcon.

### Windows via Intune

1. Package the `windows/` directory as a `.intunewin` file using the [IntuneWinAppUtil](https://github.com/microsoft/Microsoft-Win32-Content-Prep-Tool).
2. Create a new Win32 app in Intune.
3. Install command:
   ```
   powershell.exe -ExecutionPolicy Bypass -File "Invoke-AppDeployToolkit.ps1" -DeploymentType Install -DeployMode Interactive
   ```
4. Detection rule: Registry key `HKLM\SYSTEM\CurrentControlSet\Services\CSFalconService` exists.

### Windows via GPO

1. Place the package on a network share accessible to target machines.
2. Create a GPO with a Startup Script pointing to:
   ```
   powershell.exe -ExecutionPolicy Bypass -File "\\server\share\edr-migration\Invoke-AppDeployToolkit.ps1" -DeploymentType Install -DeployMode Silent
   ```

## Migration Flow

### macOS

```
Pre-flight checks
  ├── SwiftDialog installed?
  ├── SentinelOne installed?
  ├── CrowdStrike PKG accessible?
  └── Configuration valid?
       │
Welcome dialog (user can postpone)
       │
Progress dialog (real-time updates)
  ├── 10%  Preparing migration
  ├── 25%  Stopping SentinelOne services
  ├── 40%  Uninstalling SentinelOne
  ├── 50%  Verifying SentinelOne removal
  ├── 60%  Installing CrowdStrike Falcon
  ├── 75%  Configuring CrowdStrike (CID)
  ├── 85%  Starting CrowdStrike services
  ├── 90%  Verifying installation
  └── 100% Migration complete
       │
Success / Failure dialog
```

### Windows

```
Pre-Installation
  ├── Show welcome dialog
  ├── Check SentinelOne installed
  ├── Check CrowdStrike not already installed
  └── Balloon notification
       │
Installation
  ├── Show progress dialog
  ├── Uninstall SentinelOne (EXE/MSI/sentinelctl)
  ├── Verify SentinelOne removal
  ├── Install CrowdStrike Falcon
  └── Wait for CSFalconService
       │
Post-Installation
  ├── Verify CSFalconService running
  └── Show completion dialog
```

## Logging

| Platform | Log Location |
|---|---|
| macOS | `/var/log/grubhub_edr_migration.log` |
| Windows | `C:\Logs\GrubhubIT\EDR_Migration.log` |
| Windows (PSADT) | `C:\Logs\GrubhubIT\Grubhub_EDR_Migration.log` |

All logs include timestamps, severity levels (INFO/WARN/ERROR), and detailed output from each step.

## Troubleshooting

### SentinelOne won't uninstall

- **Anti-tamper enabled**: Ensure the `SENTINELONE_PASSPHRASE` is correct. You can find it in the SentinelOne console under **Sentinels > [endpoint] > Actions > Show Passphrase**.
- **Service stuck**: Try rebooting the endpoint and re-running the migration.
- **Manual removal**: Use `sentinelctl uninstall --passphrase <passphrase>` from a terminal/command prompt.

### CrowdStrike won't install

- **Installer not found**: Verify the path in `CROWDSTRIKE_PKG_PATH` (macOS) or `$CrowdStrikeInstallerPath` (Windows).
- **Invalid CID**: Confirm the CID is correct and includes the checksum. Format: `XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX-XX`.
- **Kernel extension (macOS)**: On macOS, the Falcon sensor may require a kernel extension or system extension approval. Pre-approve it via an MDM profile (PPPC/TCC payload).
- **Tamper protection conflict**: If SentinelOne's kernel extension is still loaded after uninstall, reboot before installing CrowdStrike.

### SwiftDialog not showing (macOS)

- Verify it's installed: `ls -la /usr/local/bin/dialog`
- Verify it runs: `/usr/local/bin/dialog --title "Test" --message "Hello"`
- Install it:
  ```bash
  curl -L "https://github.com/swiftDialog/swiftDialog/releases/latest/download/dialog-2.5.2-4777.pkg" \
    -o /tmp/dialog.pkg && sudo installer -pkg /tmp/dialog.pkg -target /
  ```

### PSADT dialogs not showing (Windows)

- Ensure the script is running in **Interactive** mode (`-DeployMode Interactive`).
- If deploying via SCCM, ensure the deployment is set to run with user interaction.
- For silent deployments, the script will still run — only the UI dialogs are suppressed.

### CrowdStrike service not starting (Windows)

- Check the service: `Get-Service CSFalconService`
- Check the event log: `Get-EventLog -LogName Application -Source CSFalcon* -Newest 10`
- The service may take up to 60 seconds to initialize after installation.

## Security Considerations

- **Passphrase handling**: The SentinelOne passphrase and CrowdStrike CID are sensitive values. Do not hardcode them in scripts stored in source control. Use your MDM's secret/variable management (Fleet secrets, SCCM task sequence variables, Intune script parameters, etc.).
- **Logging**: Passphrases are not logged in plaintext. Review log files to ensure no secrets are inadvertently captured.
- **Network access**: The CrowdStrike sensor will need outbound access to `*.crowdstrike.com` on ports 443 to communicate with the Falcon cloud.

## Exit Codes

| Code | Meaning |
|---|---|
| `0` | Migration completed successfully |
| `1` | Migration failed (check logs for details) |
| `3010` | Migration completed but a reboot is required (Windows only) |

## Support

For questions or issues with this migration toolkit, contact **Grubhub IT**:

- **Email**: it-security@grubhub.com
- **Portal**: https://grubhub.service-now.com
