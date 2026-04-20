# Windows Autopilot

## Reference links
- [Windows MDM Setup](https://fleetdm.com/guides/windows-mdm-setup#windows-autopilot)
- [Autopilot add devices](https://learn.microsoft.com/en-us/autopilot/add-devices)
- [Assigning Intune licenses](https://learn.microsoft.com/en-gb/intune/intune-service/fundamentals/licenses-assign)
- [Serve locally built Fleetd during Autopilot](https://github.com/fleetdm/fleet/blob/main/docs/Contributing/getting-started/testing-and-local-development.md#building-and-serving-your-own-fleetd-basemsi-installer-for-windows)

## License requirements

Each user who enrolls a device needs **both** of the following:
- **Microsoft Intune Plan 1** (for Autopilot device registration)
- **Microsoft Entra ID P1** (for automatic MDM enrollment)

The simplest option is **Enterprise Mobility + Security E3**, which bundles both.

To assign licenses:
1. Go to [Microsoft 365 Admin Center](https://admin.microsoft.com) > Users > Active users > select your user
2. Click **Licenses and apps** and assign the license
3. Make sure **Usage location** is set (e.g., United States) -- license assignment fails without it
4. If no licenses are available, check if other developers have unused ones that can be reassigned. Purchasing new licenses will be charged on Noah Talerman's (as of 24th February 2026) Brex card

**Important:** The license must be assigned to the **user who signs in during enrollment**, not just to an admin account. Without a license, the MDM URL will be empty in `dsregcmd /status` and enrollment will fail silently.

## Entra tenant configuration

### Verify Mobility (MDM and WIP) app settings

In [entra.microsoft.com](https://entra.microsoft.com) > search "Mobility" > **Mobility (MDM and WIP)**:

- **Fleet app**: MDM user scope must be **All** or **Some** (for a group containing your test users). If this is set to None, Entra will not issue an MDM URL to any user and enrollment will fail.
- **Microsoft Intune app**: MDM user scope must be **None**. If both Fleet and Intune have active scopes, devices may enroll in Intune instead of Fleet.

### Verify Entra app configuration

Follow the [Windows MDM Setup guide](https://fleetdm.com/guides/windows-mdm-setup#step-2-connect-fleet-to-microsoft-entra-id) to configure the Fleet app in Entra, including API permissions, Application ID URI, and admin consent.

## Configuring Windows Autopilot for development

To set up Windows Autopilot for development, follow these steps:
1. Create a [new Intune security group](https://intune.microsoft.com/#view/Microsoft_AAD_IAM/AddGroupBlade)
    1. Name the group
    2. Select "Dynamic Device" as the membership type
    3. Add the following dynamic query, by clicking "Add dynamic query" and "Edit" on the Rule syntax box:
        1. `(device.devicePhysicalIds -any _ -eq "[OrderID]:<YOUR_GROUP_TAG>")`
        2. Replace `<YOUR_GROUP_TAG>` with a unique identifier for your group, such as "NameDev"
2. Create a new [Autopilot deployment profile](https://intune.microsoft.com/#view/Microsoft_Intune_Enrollment/AutopilotDeploymentProfiles.ReactView) with the following settings:
    1. A name, and "Convert all targeted devices to Autopilot" set to "No"
    2. Deployment mode set to "User-driven"
    3. The rest can be the default settings
    4. On the assignments page, click "Add group" and select the security group you created in step 1.

## Recommended VM platform: Proxmox

Proxmox VE is the recommended platform for Autopilot VM testing. It provides:
- Web-based console with full interactive OOBE access (critical for Autopilot testing)
- Snapshot support (via LVM-Thin or ZFS storage)
- x86_64 native performance via KVM

### Known issue: cross-tenant hash collisions

Autopilot device matching is global across all Microsoft tenants. If another organization has registered a generic QEMU/Q35 VM hash in their Autopilot, your Proxmox VM may be claimed by their tenant during OOBE. To minimize this, maximize VM hardware uniqueness (see below).

### Creating the VM

In the Proxmox web UI, click **Create VM**:

**General:** Name your VM, note the VM ID (referred to as `<VMID>` below)

**OS:**
- ISO image: Windows 11 ISO (latest recommended -- fewer updates during OOBE)
- Type: Microsoft Windows, Version: 11/2022/2025
- Check "Add additional drive for VirtIO drivers" and select [VirtIO drivers ISO](https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/stable-virtio/virtio-win.iso)

**System:**
- Machine: **q35** (required for TPM)
- BIOS: **OVMF (UEFI)**
- Add EFI Disk: checked, storage: **LVM-Thin** (required for snapshots)
- TPM: checked, storage: **LVM-Thin**, version: **v2.0**
- SCSI Controller: **VirtIO SCSI single**
- Pre-Enroll keys: **checked** (enables Secure Boot with Microsoft keys)
- QEMU Agent: **unchecked**

**Disks:**
- Bus/Device: **SCSI**, Storage: **LVM-Thin**, Size: **40 GB**

**CPU:** Cores: 2, Type: **host**

**Memory:** 8192 MB

**Network:** Model: **Intel E1000E** (works natively in Windows without drivers -- critical for OOBE network connectivity. Do not use VirtIO for network as it requires drivers that are not available during OOBE.)

### Make the VM uniquely identifiable

SSH into the Proxmox host.

**Set unique SMBIOS values** (all fields must be alphanumeric only -- no hyphens, no periods):
```bash
qm set <VMID> -smbios1 "serial=<NAME>$(date +%s),uuid=$(cat /proc/sys/kernel/random/uuid),manufacturer=FleetDM,product=AutopilotDevVM,sku=<NAME>SKU$(date +%s),family=FleetAutopilotFamily,version=v1"
```
Replace `<NAME>` with your name (e.g., `BOB`). The `$(date +%s)` timestamp ensures global uniqueness.

**Verify:**
```bash
qm config <VMID> | grep smbios
```

### Install Windows 11

1. Start the VM, open the Console in Proxmox web UI
2. Boot from the Windows 11 ISO
3. At disk selection ("Where do you want to install Windows?"), the disk won't be visible because it uses VirtIO SCSI
4. Click **Load driver** > Browse > VirtIO CD > `vioscsi\w11\amd64` > OK > select the driver > Next
5. The disk appears. Select it and continue
6. Choose **Windows 11 Pro**, select "I don't have a product key"
7. Complete installation, let it reboot
8. Internet is not needed during installation

## Enrolling the device

### Register with Autopilot

At the OOBE screen, press **Shift+F10** to open a command prompt:

1. Verify network: `ping 8.8.8.8` (if this fails, run `ipconfig /renew`)

2. Register directly with Intune (preferred):
```powershell
powershell
$DebugPreference = "Continue"
$VerbosePreference = "Continue"
Set-ExecutionPolicy Bypass -Scope Process -Force
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
Install-PackageProvider -Name NuGet -MinimumVersion 2.8.5.201 -Force
Install-Script -Name Get-WindowsAutopilotInfo -Force
Get-WindowsAutopilotInfo -Online
```
The `$DebugPreference` and `$VerbosePreference` settings enable diagnostic output that helps troubleshoot issues with `Get-WindowsAutopilotInfo -Online`.

Sign in with your Entra account when prompted.

3. If `-Online` fails (auth errors, script errors), fall back to CSV:
```powershell
Get-WindowsAutopilotInfo -OutputFile C:\AutopilotHash.csv
```
Transfer the CSV and import it manually in Intune at Devices > Windows > Enrollment > Windows Autopilot > Devices > Import.

> **Important:** When uploading the hardware hash CSV, include the **group tag** that matches your dynamic security group query (e.g., `NameDev`). If you forget, you can edit the device in the [Autopilot devices list](https://intune.microsoft.com/#view/Microsoft_Intune_Enrollment/AutopilotDevices.ReactView/filterOnManualRemediationRequired~/false) and add it later.

### Wait for profile assignment

In [intune.microsoft.com](https://intune.microsoft.com) > Devices > Windows > Enrollment > Windows Autopilot > Devices:
- Wait until your device shows **Profile status: Assigned** (15-40 min)
- Click on the device and verify **Assigned profile** shows your deployment profile name
- Do not proceed until both are confirmed

### Sysprep and snapshot

Sysprep resets OOBE while preserving UEFI state (including the Autopilot marker). BitLocker must be off first.

Back in the VM command prompt:

1. Disable BitLocker (Windows may auto-enable it during install):
```
manage-bde -off C:
```
Wait for decryption to finish. Check with `manage-bde -status C:` -- look for "Conversion Status: Fully Decrypted".

2. Run sysprep:
```
c:\windows\system32\sysprep\sysprep.exe /generalize /oobe /shutdown
```

3. After the VM shuts down, take a snapshot:
```bash
qm snapshot <VMID> clean-oobe
```

### Test Autopilot enrollment

1. Restore snapshot and start:
```bash
qm rollback <VMID> clean-oobe
qm start <VMID>
```

2. OOBE starts. Autopilot skips region/keyboard/network screens and goes straight to checking for updates, then shows your org branding and the Entra sign-in screen.
3. Sign in with your Entra test user credentials (must have Intune + Entra P1 license assigned)
4. Device joins Entra and enrolls in Fleet MDM

### Subsequent test cycles

Restore the snapshot and start the VM:
```bash
qm rollback <VMID> clean-oobe
qm start <VMID>
```

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| OOBE shows unfamiliar org logo | Cross-tenant hash collision: another org registered a similar QEMU VM | Delete their registration or maximize VM uniqueness |
| "Name your device" instead of Autopilot branding | Profile not assigned, or AUTOPILOT_MARKER missing | Verify Profile status = Assigned in Intune; re-register and re-sysprep |
| `ZtdDeviceIsNotRegistered` in diagnostics (Ctrl+Shift+D) | Device hash not registered in any tenant | Upload hash to Intune and wait for Assigned |
| `binarySecurityToken is empty` in Fleet logs | User's Intune/Entra P1 license not assigned, or Fleet MDM user scope = None | Assign license to the enrolling user; set Fleet MDM user scope to All in Entra Mobility |
| `0x80180024` error during enrollment | Authentication failure -- often stale state from failed attempts | Restore clean snapshot and retry |
| MDM URL empty in `dsregcmd /status` | License not assigned or not yet propagated, or Fleet MDM user scope = None | Assign license, set scope to All, wait 15-30 min, run `dsregcmd /refreshprt` |
| Sysprep fails with BitLocker error | BitLocker auto-enabled during Windows install | Run `manage-bde -off C:` and wait for full decryption |
| Disk not visible during Windows install | VirtIO SCSI needs driver | Load driver from VirtIO CD: `vioscsi\w11\amd64` |
| No network in OOBE | Wrong network adapter type | Use Intel E1000E (not VirtIO) for the network adapter |
| Snapshots not available | Wrong storage backend | Must use LVM-Thin, ZFS, or qcow2 on directory storage (not regular LVM) |
| `smbios1: invalid format` error | Hyphens or periods in SMBIOS fields | Use alphanumerics only for serial, sku, and version fields |
| Scripts disabled in PowerShell | Execution policy | Run `Set-ExecutionPolicy Bypass -Scope Process -Force` |

## Verification commands (inside Windows)

Check SMBIOS values:
```powershell
Get-CimInstance Win32_ComputerSystemProduct | Select Vendor, Name, UUID, IdentifyingNumber
```

Check MDM enrollment state:
```powershell
dsregcmd /status
```
Key fields: `AzureAdJoined`, `MdmUrl`, `TenantName`

Check Autopilot diagnostics during OOBE: press **Ctrl+Shift+D**

## Setting up a custom domain with ngrok

Microsoft Entra requires a **verified custom domain** for the MDM application URIs. You cannot use a raw `*.ngrok.io` URL -- Entra will reject it during domain verification.

1. **Register a domain** (e.g., a cheap `.xyz` domain from Namecheap). You don't need to purchase SSL -- ngrok handles TLS termination.
2. **Add a subdomain in ngrok's dashboard** (Domains section) -- e.g., `mdm.yourdomain.xyz`. ngrok will provide a CNAME target (e.g., `xxx.ngrok-dns.com`).
3. **Configure DNS in your domain registrar:**
   - Add a **CNAME record** for the subdomain (e.g., `mdm`) pointing to the ngrok CNAME target.
   - Add the **TXT record** that Microsoft Entra provides on the **root domain** (e.g., `yourdomain.xyz`) for domain verification.
   - Note: DNS standards don't allow CNAME records to coexist with other record types at the same name. Using a subdomain for the CNAME avoids this conflict -- the root domain stays free for the Entra TXT verification record.
4. **Verify the root domain in Entra:** go to [Entra > Domain names](https://entra.microsoft.com/#view/Microsoft_AAD_IAM/ActiveDirectoryMenuBlade/~/Domains) > Add custom domain, enter your root domain (e.g., `yourdomain.xyz`), and verify it using the TXT record.
5. **Configure the MDM application in Entra** following the [Windows MDM Setup guide](https://fleetdm.com/guides/windows-mdm-setup#step-2-connect-fleet-to-microsoft-entra-id). Use your **subdomain** (e.g., `mdm.yourdomain.xyz`) for all MDM URLs (Application ID URI, discovery URL, terms of use URL).

**Alternative: Caddy reverse proxy.** If you have a server with a public IP and a domain you control, you can use [Caddy](https://caddyserver.com/) as a reverse proxy with automatic Let's Encrypt TLS instead of ngrok. Point a DNS A record to your server's IP and run:
```bash
caddy reverse-proxy --from your-subdomain.yourdomain.com --to https://localhost:8080
```
Caddy auto-provisions the TLS certificate. This avoids ngrok entirely and is more stable for long-running test setups.

Example ngrok config with a custom domain for the Fleet server:
```yaml
version: "3"
agent:
    authtoken: <your_ngrok_authtoken>
tunnels:
    fleet:
        proto: http
        schemes: [https]
        hostname: mdm.yourdomain.xyz  # subdomain CNAME'd to ngrok
        addr: https://localhost:8080
        inspect: true
    installers:
        proto: http
        schemes: [https]
        hostname: installers.your-ngrok-subdomain.ngrok.io
        addr: http://localhost:8085
        inspect: true
    tuf:
        proto: http
        schemes: [http]
        hostname: tuf.your-ngrok-subdomain.ngrok.io
        addr: http://localhost:8081
        inspect: true
```

Only the Fleet server tunnel needs the custom domain. The installer and TUF tunnels can use regular ngrok subdomains.
