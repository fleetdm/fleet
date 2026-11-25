# Deploy the Smallstep agent on macOS with Fleet

If you use Smallstep as your certificate authority, you can deploy the Smallstep agent to your macOS hosts through Fleet. The agent syncs richer device identity information into Smallstep than Fleet's native Smallstep certificate authority integration alone, and Smallstep generates the configuration profile for you, so you don't need to build any payloads by hand.

> **Note:** This is different from [configuring Smallstep as a certificate authority for Wi-Fi](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#smallstep) in Fleet, which delivers certificates through Fleet's SCEP proxy without installing an agent. You can use either approach on its own, or both together.

## Prerequisites

Check these before you start:

- A Smallstep team with Pro features enabled.
- macOS hosts enrolled in Fleet's MDM.
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

3. In the save modal, name the report (for example, "Smallstep enrollment"), set **Interval** to **Every hour**, and set **Target** to **macOS** and **All hosts**.
4. Select **Save**, then note the report's numeric ID. You'll need it in the next step.

> **Note:** If you already created this report while setting up Windows or Linux, enable **macOS** as an additional target on that report instead of creating a new one.

## Connect Fleet to Smallstep

1. In the Smallstep console, go to **Settings > Device Management**.
2. Under **Available Providers**, find **Fleet** and select **Connect**.
3. Fill in **API Base URL** (your Fleet server address), **API Token** (from Prerequisites), and **Enrollment Query ID** (the report ID from the previous step). Add a **Name/Alias** if you want one.
4. Select **Connect MDM**.
5. Under **Authority Certificates**, download the Root CA certificate.

Within a few minutes, your Fleet hosts appear in Smallstep's **Devices** tab. Smallstep syncs device inventory from Fleet approximately every eight hours.

## Download and upload the configuration profile

Smallstep generates a configuration profile that bundles your root and intermediate CA certificates, the agent settings, and the agent's Login Items entry, all in one file.

1. In the Smallstep console, go to **Settings > Device Management**, open your Fleet connection, and choose the **Settings** tab.
2. Under **Configuration Profile (macOS only)**, select **Download**. The file is named `fleet-<connection-id>.mobileconfig`.
3. In Fleet, go to **Controls > OS settings > Configuration profiles** and select **Add profile**.
4. Upload the `.mobileconfig` file and scope it to your macOS hosts.

The profile deploys to devices at their next MDM check-in.

## Deploy the Smallstep agent with Fleet software

1. Download the [latest macOS package](https://packages.smallstep.com/stable/darwin/step-agent_latest.pkg). Earlier versions are available at [releases.smallstep.com](https://releases.smallstep.com).
2. In Fleet, go to **Software**, select **Add software > Custom package**, and upload the package.
3. Scope the install to your macOS hosts.

The agent picks up the configuration from the profile Fleet already deployed and uses it to register the device with Smallstep.

> **Note:** You can deploy the agent with a different software management tool, such as Munki, instead of Fleet's software feature. See Smallstep's [manual installation guide](https://smallstep.com/docs/platform/smallstep-agent/#macos-installation).

## Optional: deploy the osquery extension

Smallstep's osquery extension registers additional device identity information directly in Fleet, giving enrollment a richer set of inventory data to match against.

1. Download the `step-agent-osquery` package for macOS from [Smallstep's releases](https://releases.smallstep.com).
2. In Fleet, go to **Software**, select **Add software > Custom package**, and upload the package.
3. Enable **Deploy**, then select **Add software**.

Fleet creates a policy that automatically triggers installation on hosts that don't have the package yet. See [deploying custom osquery extensions in Fleet](https://fleetdm.com/guides/deploying-custom-osquery-extensions-in-fleet-a-step-by-step-guide) for more detail and troubleshooting.

## Verify

1. In Fleet, go to **Hosts**, select a macOS host that received the profile, and check its **OS settings** status.
2. In the Smallstep console, go to **Devices** and confirm the host is enrolled with a certificate.
3. On the host, check the agent's status:

   ```bash
   /Applications/SmallstepAgent.app/Contents/MacOS/SmallstepAgent status
   ```

## Troubleshoot

**OS settings status shows the profile failed**

Check the host's **OS settings** error detail on its Fleet host page. A failed profile usually means the host hasn't checked in with MDM recently, or the profile's certificate data is stale: re-download it from Smallstep and re-upload it to Fleet.

**Host doesn't show up in Smallstep's Devices tab**

Confirm the Fleet API token used in the connection hasn't expired or been revoked, and that the Smallstep enrollment report is targeting macOS.

## Further reading

- [Connect Fleet DM to Smallstep](https://smallstep.com/docs/tutorials/connect-fleet-dm-to-smallstep/): Smallstep's own tutorial, covering Windows and Linux in addition to macOS.
- [Deploy the Smallstep agent on Windows with Fleet](https://fleetdm.com/guides/deploy-smallstep-agent-on-windows-with-fleet)
- [Connect Smallstep as a certificate authority for Wi-Fi](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#smallstep)

<meta name="articleTitle" value="Deploy the Smallstep agent on macOS with Fleet">
<meta name="authorFullName" value="kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="publishedOn" value="2026-11-30">
<meta name="category" value="guides">
<meta name="description" value="Deploy and register the Smallstep agent on macOS hosts using a Fleet configuration profile and software.">
