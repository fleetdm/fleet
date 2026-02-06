# Deploying Okta Desktop MFA for Windows with Fleet

Okta Desktop MFA (Okta Device Access) brings multi-factor authentication to the Windows login screen, lock screen, and privilege elevation prompts. Instead of only protecting web applications, Desktop MFA extends MFA protection to local Windows authentication events.

This guide shows how to deploy Okta Desktop MFA to Windows devices using Fleet.

## Why use Desktop MFA?

Desktop MFA closes a security gap by requiring MFA at the Windows login screen. Without it, an attacker with stolen credentials can log in to a Windows device without triggering MFA, even if all your web apps require it.

With Desktop MFA enabled, users authenticate with their Okta credentials and complete MFA at every Windows login, unlock, and privilege elevation event. This ensures consistent authentication security across all access points.

## Requirements

Before deploying Desktop MFA, ensure you meet these requirements:

### Windows requirements

| Requirement | Details |
|------------|---------|
| **Windows edition** | Windows 10/11 Pro, Enterprise, or Education (Home edition not supported) |
| **Domain join** | Device must be Azure AD-joined or on-premises AD-joined |
| **Windows version** | Windows 10 version 1709 (Build 16299) or later |
| **Administrator access** | Installation and policy deployment require local admin privileges |

### Okta requirements

| Requirement | Details |
|------------|---------|
| **Okta edition** | Workforce Identity Cloud with Desktop MFA capability enabled |
| **Desktop MFA application** | Created in Okta Admin Console |
| **OAuth credentials** | Client ID and Client Secret from Desktop MFA app |
| **Okta Verify version** | Latest version with Desktop MFA support |

Contact your Okta account representative if you need to purchase the Desktop MFA license for your organization.

### Fleet requirements

| Requirement | Details |
|------------|---------|
| **Fleet secrets** | Three secret variables configured for OAuth credentials |
| **Maintained app** | Okta Verify installer uploaded as Fleet software |
| **PowerShell scripts** | Install and policy scripts deployed via Fleet |
| **Policy monitoring** | osquery policy for compliance verification |

## Configure Fleet secret variables

Desktop MFA requires three Fleet secret variables to securely store OAuth credentials. These secrets keep credentials out of scripts and centralize secret management.

Configure these secrets in your Fleet server:

1. In Fleet, navigate to **Controls** → **Variables**
2. Click **Add variable**
3. Create these three secrets:

| Variable name | Example value | Description |
|--------------|--------------|-------------|
| `OKTA_DESKTOP_MFA_TENANT_URL` | `https://your-org.okta.com` | Your Okta organization URL |
| `OKTA_DESKTOP_MFA_CLIENT_ID` | `0oa1a2b3c4d5e6f7g8h9` | OAuth client ID from Desktop MFA app |
| `OKTA_DESKTOP_MFA_CLIENT_SECRET` | (84-character string) | OAuth client secret from Desktop MFA app |

## Set up Okta Desktop MFA application

Configure the Desktop MFA application in your Okta Admin Console:

1. Sign in to your Okta org as a super admin
2. Navigate to **Applications** → **Browse App Catalog**
3. Search for **Desktop MFA** and click **Add integration**
4. On the **General** tab, set the application label
5. On the **Sign on** tab, copy the **Client ID** (you'll need this for Fleet secrets)
6. On the **Sign on** tab, generate and copy the **Client Secret**
7. On the **Assignments** tab, assign the app to users or groups who will use Desktop MFA
8. Click **Save**

Download the Okta Verify installer from the Admin Console at **Settings** → **Downloads**. Don't download from the Microsoft Store, as that version lacks MDM integration features.

## Deploy Okta Verify via Fleet

Install Okta Verify on your Windows hosts using Fleet's software deployment:

1. In Fleet, select the team you want to deploy Desktop MFA to
2. Navigate to **Software** → **Add software** → **Custom package**
3. Click **Choose file** and select the Okta Verify installer you downloaded from Okta
4. For **Install script**, upload the install script below
5. For **Uninstall script**, upload the uninstall script below
6. Choose **Automatic** to have Fleet install on all hosts, or **Manual** to control installation per host

### Install script

The install script reads OAuth credentials from Fleet secrets and installs Okta Verify with Desktop MFA enabled:

```powershell
# Okta Verify Installation Script
# Installs Okta Verify with Desktop MFA capability enabled

$exeFilePath = "${env:INSTALLER_PATH}"

# Read Fleet secret variables
$oktaOrgUrl = $env:FLEET_SECRET_OKTA_DESKTOP_MFA_TENANT_URL
$oktaClientId = $env:FLEET_SECRET_OKTA_DESKTOP_MFA_CLIENT_ID
$oktaClientSecret = $env:FLEET_SECRET_OKTA_DESKTOP_MFA_CLIENT_SECRET

$exitCode = 0

try {
    # Validate required Fleet secrets are set
    $missingSecrets = @()
    if ([string]::IsNullOrWhiteSpace($oktaOrgUrl)) {
        $missingSecrets += "FLEET_SECRET_OKTA_DESKTOP_MFA_TENANT_URL"
    }
    if ([string]::IsNullOrWhiteSpace($oktaClientId)) {
        $missingSecrets += "FLEET_SECRET_OKTA_DESKTOP_MFA_CLIENT_ID"
    }
    if ([string]::IsNullOrWhiteSpace($oktaClientSecret)) {
        $missingSecrets += "FLEET_SECRET_OKTA_DESKTOP_MFA_CLIENT_SECRET"
    }

    if ($missingSecrets.Count -gt 0) {
        Write-Host "ERROR: Required Fleet secrets are not configured:" -ForegroundColor Red
        foreach ($secret in $missingSecrets) {
            Write-Host "  - $secret" -ForegroundColor Red
        }
        throw "Missing required Fleet secrets: $($missingSecrets -join ', ')"
    }

    Write-Host "Installing Okta Verify with organization configuration..."

    # Build argument list for silent installation
    # SKU=ALL enables Desktop MFA capability
    $argumentList = @(
        "/q",
        "SKU=ALL",
        "ORGURL=$oktaOrgUrl",
        "CLIENTID=$oktaClientId",
        "CLIENTSECRET=$oktaClientSecret"
    )

    # Start installation process
    $processOptions = @{
        FilePath = "$exeFilePath"
        ArgumentList = $argumentList
        PassThru = $true
        Wait = $true
        NoNewWindow = $true
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode

    Write-Host "Install exit code: $exitCode"

    if ($exitCode -eq 0) {
        Write-Host "Okta Verify installed successfully"
    }

} catch {
    Write-Host "Error during installation: $_"
    $exitCode = 1
} finally {
    Exit $exitCode
}
```

### Uninstall script

The uninstall script removes Okta Verify from Windows devices:

```powershell
# Okta Verify Uninstallation Script

$exitCode = 0

try {
    Write-Host "Uninstalling Okta Verify..."

    # Find Okta Verify in installed programs
    $uninstallKey = "HKLM:\Software\Microsoft\Windows\CurrentVersion\Uninstall\*"
    $uninstallKey64 = "HKLM:\Software\Wow6432Node\Microsoft\Windows\CurrentVersion\Uninstall\*"

    $oktaVerify = Get-ItemProperty $uninstallKey, $uninstallKey64 -ErrorAction SilentlyContinue |
        Where-Object { $_.DisplayName -like "*Okta Verify*" } |
        Select-Object -First 1

    if ($null -eq $oktaVerify) {
        Write-Host "Okta Verify not found in installed programs"
        Exit 0
    }

    $uninstallString = $oktaVerify.UninstallString

    if ([string]::IsNullOrEmpty($uninstallString)) {
        throw "Uninstall string not found for Okta Verify"
    }

    # Parse uninstall string
    if ($uninstallString -match '^"([^"]+)"(.*)$') {
        $uninstallerPath = $matches[1]
        $uninstallerArgs = $matches[2].Trim()
    } else {
        $uninstallerPath = $uninstallString.Split(' ')[0]
        $uninstallerArgs = ""
    }

    # Add silent uninstall argument
    if ($uninstallerArgs -notmatch "/silent|/quiet|/S|/s") {
        $uninstallerArgs = "/S $uninstallerArgs".Trim()
    }

    # Run uninstaller
    $processOptions = @{
        FilePath = $uninstallerPath
        PassThru = $true
        Wait = $true
        NoNewWindow = $true
    }

    if (-not [string]::IsNullOrEmpty($uninstallerArgs)) {
        $processOptions.ArgumentList = $uninstallerArgs
    }

    $process = Start-Process @processOptions
    $exitCode = $process.ExitCode

    Write-Host "Uninstall exit code: $exitCode"

} catch {
    Write-Host "Error during uninstallation: $_"
    $exitCode = 1
} finally {
    Exit $exitCode
}
```

## Configure Desktop MFA policies

After Okta Verify is installed, deploy registry policies to enforce MFA at Windows login. These policies control when and how Desktop MFA is required.

Deploy this script via Fleet to configure registry policies:

1. In Fleet, navigate to **Scripts**
2. Click **Add script**
3. Paste the policy configuration script below
4. Save and run the script on hosts that have Okta Verify installed

### Policy configuration script

```powershell
# Desktop MFA Policy Configuration Script
# Configures Windows registry policies for MFA enforcement

$RegistryPath1 = "HKLM:\Software\Policies\Okta\"
$RegistryPath2 = "HKLM:\Software\Policies\Okta\Okta Device Access"

# Create registry paths if they don't exist
If (-NOT (Test-Path $RegistryPath1)) {
    New-Item -Path $RegistryPath1 -Force | Out-Null
}
If (-NOT (Test-Path $RegistryPath2)) {
    New-Item -Path $RegistryPath2 -Force | Out-Null
}

# Configure Desktop MFA policies
New-ItemProperty -Path $RegistryPath2 -Name 'MFARequiredList' -PropertyType MultiString -Value ('*') -Force
New-ItemProperty -Path $RegistryPath2 -Name "MaxLoginsWithoutEnrolledFactors" -PropertyType DWord -Value 0 -Force
New-ItemProperty -Path $RegistryPath2 -Name "MFAGracePeriodInMinutes" -PropertyType DWord -Value 0 -Force

Write-Output "Desktop MFA policies configured successfully"
```

This configuration:

- **MFARequiredList:** `*` requires all users to authenticate with MFA
- **MaxLoginsWithoutEnrolledFactors:** `0` forces immediate MFA enrollment
- **MFAGracePeriodInMinutes:** `0` requires MFA at every login with no grace period

Policy changes take effect within 10 minutes after the script runs.

## Monitor compliance and enforce with automation

Fleet can automatically enforce Desktop MFA configuration using policies and automations. The policy checks for both Okta Verify installation and registry configuration. When a host fails the policy, Fleet automation triggers the policy configuration script to remediate.

### Create the compliance policy

Create a Fleet policy to monitor Desktop MFA deployment across your Windows hosts:

1. In Fleet, navigate to **Policies**
2. Click **Add policy**
3. Give the policy a name like "Okta Desktop MFA configured"
4. Paste this query:

```sql
SELECT 1 FROM programs
  WHERE
    name LIKE '%Okta Verify%'
    AND EXISTS (
      SELECT 1
      FROM registry
      WHERE path = 'HKEY_LOCAL_MACHINE\Software\Policies\Okta\Okta Device Access\MFARequiredList'
    );
```

This policy checks two conditions:

- Okta Verify is installed (from `programs` table)
- MFA registry policies are configured (checks `MFARequiredList` key exists)

The policy returns passing only when both conditions are true.

1. Save the policy

### Configure automation for self-healing

Now configure Fleet automation to run the policy configuration script when hosts fail the policy:

1. In Fleet, navigate to **Policies**
2. Find the "Okta Desktop MFA configured" policy you just created
3. Click the policy name to open policy details
4. Click **Manage automations**
5. Under **Run script**, select the MFA policy configuration script you uploaded earlier
6. Save the automation

With this automation configured, Fleet will:

1. Run the compliance policy on its schedule
2. Detect hosts where Okta Verify is installed but registry policies are missing
3. Automatically run the policy configuration script on failing hosts
4. Re-check compliance on the next policy run

This creates a self-healing enforcement loop. If registry policies are removed or a host is reimaged with Okta Verify but missing policies, Fleet will automatically reconfigure them.

## End user experience

When Desktop MFA is deployed to a Windows host, users see these prompts at their next login or lock screen event:

1. **Initial enrollment:** Users are prompted to scan a QR code with the Okta Verify mobile app
2. **Factor setup:** Users configure their preferred MFA method (push notification, biometric, etc.)
3. **Login authentication:** At every Windows login or unlock, users enter their username and complete MFA
4. **Privilege elevation:** When UAC prompts appear, users complete MFA to elevate privileges

After enrollment, Desktop MFA is active at every Windows authentication event. Users can no longer bypass MFA by directly accessing their device.

## Troubleshooting

### Desktop MFA not prompting at login

Check these items:

- Device is domain-joined (run `dsregcmd /status` and verify domain join status)
- Okta Verify installed with `SKU=ALL` parameter
- Registry policies exist at `HKLM:\Software\Policies\Okta\Okta Device Access`
- Wait 10 minutes after policy deployment for changes to propagate

### Installation fails with missing secrets error

Check these items:

- All three Fleet secrets are configured in **Controls** → **Variables**
- Secret names match exactly (case-sensitive): `OKTA_DESKTOP_MFA_TENANT_URL`, `OKTA_DESKTOP_MFA_CLIENT_ID`, `OKTA_DESKTOP_MFA_CLIENT_SECRET`
- OAuth credentials are correct (Client ID and Secret from Okta Desktop MFA app)

### Policy not detecting configuration

Verify the registry key exists:

1. Open Registry Editor on the Windows host
2. Navigate to `HKEY_LOCAL_MACHINE\Software\Policies\Okta\Okta Device Access`
3. Confirm `MFARequiredList` key exists with value `*`

If the key is missing, re-run the policy configuration script.

## Additional resources

For more information about Okta Desktop MFA configuration and troubleshooting, see the [official Okta Desktop MFA documentation](https://help.okta.com/oie/en-us/content/topics/oda/windows-mfa/deploy-win-mfa.htm).

To learn more about Fleet's software deployment and script execution capabilities, see the [Fleet documentation](https://fleetdm.com/docs).

[*Get started with Fleet*](https://fleetdm.com/docs/get-started/why-fleet)

<meta name="category" value="guides">
<meta name="authorGitHubUsername" value="tux234">
<meta name="authorFullName" value="Mitch Francese">
<meta name="publishedOn" value="2026-02-06">
<meta name="articleTitle" value="Deploying Okta Desktop MFA for Windows">
<meta name="description" value="Learn how to deploy Okta Desktop MFA to Windows devices using Fleet MDM">
