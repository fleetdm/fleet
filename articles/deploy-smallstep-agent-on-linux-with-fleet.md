# Deploy the Smallstep agent on Linux with Fleet

If you use Smallstep as your certificate authority, you can deploy the Smallstep agent to your Linux hosts through Fleet. Linux doesn't support MDM configuration profiles, so there's no certificate authority to configure in Fleet here: the agent registers directly with your Smallstep team using TPM attestation.

## Prerequisites

Check these before you start:

- A Smallstep team with Pro features enabled.
- Linux hosts enrolled in Fleet, with a TPM.
- Devices that can reach `smallstep.com`, `api.smallstep.com`, `gateway.smallstep.com`, `control.infra.smallstep.com`, `*.<team-name>.ca.smallstep.com`, `auth.smallstep.com`, `att.smallstep.com`, and `tel.smallstep.com`.
- A Fleet API token. Use a personal token for testing. For production, create a dedicated API-only user instead:

  ```bash
  fleetctl user create --name 'Smallstep' --email 'smallstep-api@example.com' --password 'your-secure-password' --api-only
  ```

  The command prints the new user's API token. The default **Observer** role is enough for Smallstep's device sync.

## Deploy the Smallstep agent with Fleet software

Each package needs a post-install script that writes the agent's configuration and starts the service.

1. Save the following as `smallstep-agent-config.sh`, replacing `<your-team-slug>` and `<your-agents-ca-fingerprint>` with your team slug (**Settings > Team** in Smallstep) and the SHA-256 root fingerprint of your Smallstep Agents authority (**Certificate Manager > Authorities** in Smallstep, under the Agents authority):

   ```bash
   #!/bin/bash
   mkdir -p /etc/step-agent
   cat > /etc/step-agent/agent.yaml <<EOF
   team: "<your-team-slug>"
   fingerprint: "<your-agents-ca-fingerprint>"
   EOF
   systemctl daemon-reload
   systemctl enable --now step-agent
   ```

2. Download the package for your distribution and architecture: [Debian/Ubuntu x64](https://packages.smallstep.com/stable/linux/step-agent_amd64_latest.deb), [Debian/Ubuntu arm64](https://packages.smallstep.com/stable/linux/step-agent_arm64_latest.deb), [RHEL/Fedora x64](https://packages.smallstep.com/stable/linux/step-agent_x86_64_latest.rpm), or [RHEL/Fedora arm64](https://packages.smallstep.com/stable/linux/step-agent_aarch64_latest.rpm). Earlier versions are available at [releases.smallstep.com](https://releases.smallstep.com).
3. In Fleet, go to **Software**, select **Add software > Custom package**, and upload the package. Attach the post-install script from step 1.
4. Scope the install to the hosts that match that package, using [labels](https://fleetdm.com/guides/managing-labels-in-fleet) to target `.deb` packages to Debian/Ubuntu hosts and `.rpm` packages to RHEL/Fedora hosts.

If your fleet spans multiple distributions or architectures, repeat steps 2-4 for each variant.

On startup, the agent reads `/etc/step-agent/agent.yaml`, then attests with the host's TPM.

## Optional: deploy the osquery extension

Smallstep's osquery extension registers additional device identity information, including the TPM endorsement key, directly in Fleet.

1. Download the `step-agent-osquery` package for your distribution and architecture from [Smallstep's releases](https://releases.smallstep.com).
2. In Fleet, go to **Software**, select **Add software > Custom package**, and upload the package.
3. Enable **Deploy**, then select **Add software**.

Fleet creates a policy that automatically triggers installation on hosts that don't have the package yet. This policy has no minimum version requirement; to force a newer version onto hosts, edit the policy to add a condition like `and version >= '0.65.1-1'`. See [deploying custom osquery extensions in Fleet](https://fleetdm.com/guides/deploying-custom-osquery-extensions-in-fleet-a-step-by-step-guide) for more detail and troubleshooting.

## Create a Smallstep enrollment report in Fleet

Once the osquery extension is active, create a report so Smallstep can pull in the TPM endorsement key and other enrollment details alongside Fleet's standard inventory.

1. Go to **Reports** and select **Add report**.
2. Enter the query:

   ```sql
   SELECT * FROM smallstep_enrollment;
   ```

3. In the save modal, name the report (for example, "Smallstep enrollment"), set **Interval** to **Every hour**, and set **Target** to **Linux** and **All hosts**.
4. Select **Save**, then note the report's numeric ID.

> **Note:** If you already created this report while setting up Windows or macOS, enable **Linux** as an additional target on that report instead of creating a new one.

## Connect Fleet to Smallstep

1. In the Smallstep console, go to **Settings > Device Management**.
2. Under **Available Providers**, find **Fleet** and select **Connect** (or, if you already connected Fleet for another platform, open that connection and edit it instead).
3. Fill in **API Base URL** (your Fleet server address), **API Token** (from Prerequisites), and **Enrollment Query ID** (the report ID from the previous step). Add a **Name/Alias** if you want one.
4. Select **Connect MDM**.

Within a few minutes, your Fleet hosts appear in Smallstep's **Devices** tab. Smallstep syncs device inventory from Fleet approximately every four hours.

By default, devices synced from Fleet need manual approval: hosts show as pending in the Smallstep console until you approve them.

## Optional: approve devices automatically

To skip manual approval, add `Fleet` to `autoApproveSources` in your team's device enrollment policy. This isn't exposed in the Smallstep console, so use the [device enrollment policy API](https://smallstep.com/docs) with a token scoped to `get-device-enrollment-policy` and `put-device-enrollment-policy`. Fetch the current policy first, since the update replaces it entirely:

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

On a Fleet-enrolled Linux test host:

1. Confirm the agent service is running:

   ```bash
   sudo systemctl status step-agent
   ```

2. Confirm certificate files are present under `/var/lib/step-agent`.
3. In the Smallstep console, go to **Devices** and confirm the host (approve it if it's still pending).
4. In the Smallstep console, go to **Certificate Manager > Inventory** and confirm a freshly issued certificate matching the device.

## Troubleshoot

**Host stays pending in the Smallstep console**

Devices synced from Fleet need manual approval by default. Approve the host in the Smallstep console, or set up automatic approval (see [Optional: approve devices automatically](#optional-approve-devices-automatically)).

**Agent service isn't running**

Confirm the software install succeeded on the host's **Software** tab in Fleet, and that the post-install script ran without error. A missing or malformed `/etc/step-agent/agent.yaml` is the most common cause: check the file against step 1 under [Deploy the Smallstep agent with Fleet software](#deploy-the-smallstep-agent-with-fleet-software).

**No TPM available**

The agent needs a TPM to attest with Smallstep. Hosts without one can't complete enrollment through this flow.

## Further reading

- [Connect Fleet DM to Smallstep](https://smallstep.com/docs/tutorials/connect-fleet-dm-to-smallstep/): Smallstep's own tutorial, covering macOS and Windows in addition to Linux.
- [Deploy the Smallstep agent on macOS with Fleet](https://fleetdm.com/guides/deploy-smallstep-agent-on-macos-with-fleet)
- [Deploy the Smallstep agent on Windows with Fleet](https://fleetdm.com/guides/deploy-smallstep-agent-on-windows-with-fleet)

<meta name="articleTitle" value="Deploy the Smallstep agent on Linux with Fleet">
<meta name="authorFullName" value="kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="publishedOn" value="2026-11-30">
<meta name="category" value="guides">
<meta name="description" value="Deploy and register the Smallstep agent on Linux hosts using Fleet's software feature and TPM attestation.">
