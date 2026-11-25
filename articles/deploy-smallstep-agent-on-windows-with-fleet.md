# Deploy the Smallstep agent on Windows with Fleet

Fleet's built-in Smallstep certificate authority integration currently covers macOS, iOS, and iPadOS only. Until Fleet adds native Smallstep support for Windows, you can bring Windows hosts into the same Smallstep team by deploying the Smallstep agent directly, using Fleet's software and scripts features. The agent registers with Smallstep using TPM attestation, so no certificate authority setup is required in Fleet.

## Prerequisites

Check these before you start:

- A Smallstep team with Pro features enabled.
- Windows hosts enrolled in Fleet's MDM, running Windows 10 or later Pro or Enterprise (Windows Home isn't supported).
- Devices that can reach `smallstep.com`, `api.smallstep.com`, `gateway.smallstep.com`, `control.infra.smallstep.com`, `*.<team-name>.ca.smallstep.com`, `auth.smallstep.com`, `att.smallstep.com`, and `tel.smallstep.com`.
- A Fleet API token. Use a personal token for testing. For production, create a dedicated API-only user instead:

  ```bash
  fleetctl user create --name 'Smallstep' --email 'smallstep-api@example.com' --password 'your-secure-password' --api-only
  ```

  The command prints the new user's API token. The default **Observer** role is enough for Smallstep's device sync.

## Create a Smallstep enrollment report in Fleet

Smallstep reads this report during device sync to pull in device identity details alongside Fleet's standard inventory.

1. Go to **Reports** and select **Add report**.
2. Enter the query:

   ```sql
   SELECT * FROM smallstep_enrollment;
   ```

3. In the save modal, name the report (for example, "Smallstep enrollment"), set **Interval** to **Every hour**, and set **Target** to **Windows** and **All hosts**.
4. Select **Save**, then note the report's numeric ID. You'll need it in the next step.

> **Note:** `smallstep_enrollment` is provided by Smallstep's osquery extension. If you haven't deployed that extension to your Windows hosts, the report returns no rows until you do. See [deploying custom osquery extensions in Fleet](https://fleetdm.com/guides/deploying-custom-osquery-extensions-in-fleet-a-step-by-step-guide) for how to add it.

## Connect Fleet to Smallstep

1. In the Smallstep console, go to **Settings > Device Management**.
2. Under **Available Providers**, find **Fleet** and select **Connect**.
3. Fill in **API Base URL** (your Fleet server address), **API Token** (from Prerequisites), and **Enrollment Query ID** (the report ID from the previous step). Add a **Name/Alias** if you want one.
4. Select **Connect MDM**.
5. Under **Authority Certificates**, copy the SHA-256 fingerprint of the Agents Root CA certificate. You'll need it in the next step.

Within a few minutes, your Fleet hosts appear in Smallstep's **Devices** tab. Smallstep syncs device inventory from Fleet approximately every four hours.

## Configure the agent with a Fleet script

The Windows Smallstep agent reads its team and certificate authority from the registry, at `HKLM:\Software\Policies\Smallstep`. Fleet doesn't manage the registry directly, so a script sets these values before the agent starts.

1. Save the following as `smallstep-agent-config.ps1`, replacing `<team-slug>` and `<agents-authority-root-sha256-fingerprint>` with your team slug (**Settings > Team** in Smallstep) and the fingerprint you copied in the previous step:

   ```powershell
   $RegistryPath = "HKLM:\Software\Policies\Smallstep"
   if (-not (Test-Path $RegistryPath)) {
       New-Item -Path $RegistryPath -Force | Out-Null
   }
   Set-ItemProperty -Path $RegistryPath -Name "TeamSlug" -Value "<team-slug>"
   Set-ItemProperty -Path $RegistryPath -Name "CAFingerprint" -Value "<agents-authority-root-sha256-fingerprint>"
   ```

2. In Fleet, go to **Controls > Scripts** and select **Upload script**.
3. Upload `smallstep-agent-config.ps1`, scope it to your Windows hosts, and run it.

> **Note:** To keep these registry values in place if something resets them, pair this script with a [policy automation](https://fleetdm.com/guides/policy-automation-run-script) that checks for the registry key and reruns the script when it's missing.

## Deploy the Smallstep agent with Fleet software

1. Download the agent installer for your architecture: [x64](https://packages.smallstep.com/stable/windows/step-agent_amd64_latest.msi) or [arm64](https://packages.smallstep.com/stable/windows/step-agent_arm64_latest.msi). Earlier versions are available at [releases.smallstep.com](https://releases.smallstep.com).
2. In Fleet, go to **Software**, select **Add software > Custom package**, and upload the MSI.
3. Scope the install to your Windows hosts.

On startup, the agent reads the registry values from the previous step, then attests with the host's TPM. Smallstep matches that attestation to the device inventory synced from Fleet.

## Optional: approve devices automatically

By default, devices synced from Fleet need manual approval in Smallstep: hosts show as pending in the Smallstep console until you approve them. To skip that step, add `Fleet` to `autoApproveSources` in your team's device enrollment policy. This isn't exposed in the Smallstep console, so use the [device enrollment policy API](https://smallstep.com/docs) with a token scoped to `get-device-enrollment-policy` and `put-device-enrollment-policy`. Fetch the current policy first, since the update replaces it entirely:

```bash
curl -s --request GET \
  --url https://gateway.smallstep.com/api/device-enrollment-policy \
  --header "Authorization: Bearer $SMALLSTEP_API_TOKEN" \
  --header 'Accept: application/json'
```

Then resubmit it with `Fleet` added to `autoApproveSources`, adjusting the other fields to match what you fetched:

```bash
curl -s --request PUT \
  --url https://gateway.smallstep.com/api/device-enrollment-policy \
  --header "Authorization: Bearer $SMALLSTEP_API_TOKEN" \
  --header 'Accept: application/json' \
  --header 'Content-Type: application/json' \
  --data '{
    "allowedSources": ["Smallstep API", "Smallstep Agent", "Fleet"],
    "autoApproveSources": ["Fleet"],
    "requireUserBinding": true
  }'
```

## Verify

On a Fleet-enrolled Windows test host:

1. Confirm the registry values were written:

   ```powershell
   Get-ItemProperty -Path HKLM:\Software\Policies\Smallstep
   ```

2. Confirm the agent service is running:

   ```powershell
   sc query "Smallstep Agent"
   ```

3. In the Smallstep console, go to **Devices** and confirm the host shows an **Enrolled At** timestamp.
4. In the Smallstep console, go to **Certificate Manager > Inventory** and confirm a freshly issued certificate matching the device.

## Troubleshoot

**Host stays pending in the Smallstep console**

Devices synced from Fleet need manual approval by default. Approve the host in the Smallstep console, or set up automatic approval (see [Optional: approve devices automatically](#optional-approve-devices-automatically)).

**Registry values are empty or missing**

Confirm the script ran successfully on the host. Check **Controls > Scripts** in Fleet for the run's status and output, and rerun it if needed.

**Agent service isn't running**

Confirm the software install succeeded on the host's **Software** tab in Fleet. If the install succeeded but the service won't start, check **Event Viewer > Applications and Services Logs > Microsoft > Windows > DeviceManagement-Enterprise-Diagnostics-Provider > Admin** on the host for the underlying error.

## Further reading

- [Connect Fleet DM to Smallstep](https://smallstep.com/docs/tutorials/connect-fleet-dm-to-smallstep/): Smallstep's own tutorial, covering macOS and Linux in addition to Windows.
- [Deploy the Smallstep agent on macOS with Fleet](https://fleetdm.com/guides/deploy-smallstep-agent-on-macos-with-fleet)
- [Deploy the Smallstep agent on Linux with Fleet](https://fleetdm.com/guides/deploy-smallstep-agent-on-linux-with-fleet)
- [Deploying custom osquery extensions in Fleet](https://fleetdm.com/guides/deploying-custom-osquery-extensions-in-fleet-a-step-by-step-guide)
- [fleetdm/fleet#48925](https://github.com/fleetdm/fleet/issues/48925): tracks native Smallstep SCEP proxy support for Windows.

<meta name="articleTitle" value="Deploy the Smallstep agent on Windows with Fleet">
<meta name="authorFullName" value="kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="publishedOn" value="2026-11-30">
<meta name="category" value="guides">
<meta name="description" value="Deploy and register the Smallstep agent on Windows hosts using Fleet's software and scripts features.">
