# Deploy Cloudflare One Agent on Android

Use Fleet to install the Cloudflare One Agent on fully managed Android devices, tunnel only the apps you choose (per-app VPN), keep the VPN on at all times, and apply Cloudflare Gateway DNS policies. You can do each step in the Fleet UI or with GitOps.

This guide covers fully managed devices. Work profile (BYOD) devices, HTTP filtering, and device posture weren't tested.

## Prerequisites

- Fleet Premium with [Android Enterprise connected](https://fleetdm.com/guides/android-mdm-setup).
- A fully managed Android device enrolled in the fleet you're configuring. Factory reset the device, then scan the QR code from the fleet's enrollment page.
- A Cloudflare Zero Trust organization.
- For GitOps: a [Fleet GitOps repo](https://fleetdm.com/docs/configuration/yaml-files).

> **Warning:** Enrolling devices with a Cloudflare service token doesn't work yet. Fleet doesn't substitute secret variables (`$FLEET_SECRET_*`) in Android app managed configurations, so the agent receives the literal variable name as the credential. Don't put the token in the configuration as plaintext. Until [fleetdm/fleet#55512](https://github.com/fleetdm/fleet/issues/55512) is resolved, have users sign in with a one-time PIN or your identity provider.

## Configure Cloudflare Zero Trust

1. In Cloudflare, create your Zero Trust organization. Your team name becomes `<team>.cloudflareaccess.com`.
2. Add a device enrollment permissions rule with the action **Allow** that includes the users you want to enroll. A one-time PIN is enough, so you don't need an identity provider to test.
3. Leave the default device profile as is (WARP mode, auto-connect off), or set it to match your policy.

## Write the managed configuration

The agent reads its settings from a managed configuration. Fleet uses a plain JSON object under `managedConfiguration`. Save this as `cloudflare-one-agent.json` and replace `<your-team-name>` with your Zero Trust team name:

```json
{
  "managedConfiguration": {
    "app_config_bundle_list": [
      {
        "organization": "<your-team-name>",
        "display_name": "Fleet testing",
        "service_mode": "warp",
        "switch_locked": true,
        "auto_connect": 0,
        "support_url": "https://example.com/support",
        "use_in_app_webview": true,
        "tunneled_apps": [
          { "app_identifier": "com.android.chrome", "is_browser": true },
          { "app_identifier": "org.mozilla.firefox", "is_browser": true }
        ]
      }
    ]
  }
}
```

- `tunneled_apps` is the per-app VPN list. Only the apps listed here send traffic through Cloudflare. Add each app's package name as `app_identifier`.
- `use_in_app_webview: true` is required by Cloudflare for always-on VPN and lockdown.
- `switch_locked: true` prevents users from turning the VPN off on the device. Remove it if you want users to control the connection.

Cloudflare documents Android MDM settings only in [Intune's format](https://developers.cloudflare.com/cloudflare-one/team-and-resources/devices/cloudflare-one-client/deployment/mdm-deployment/partners/intune/) (`androidenterprise#managedConfiguration`). To translate it to Fleet's format:

1. Drop the `managedProperty` and `value*` wrappers.
2. Turn each `valueBundleArray` into a JSON array of objects.
3. Keep the settings inside `app_config_bundle_list`.

See Cloudflare's [per-app VPN parameters](https://developers.cloudflare.com/cloudflare-one/team-and-resources/devices/cloudflare-one-client/deployment/mdm-deployment/parameters/#per-app-vpn-parameters-android) for the full list of keys.

> **Note:** Android app configurations support built-in `$FLEET_VAR_*` host variables, but not `$FLEET_SECRET_*` variables. A team name isn't sensitive, so hardcoding it is fine.

## Add the agent and a browser

The agent needs a browser on the device to sign in. Add Chrome (or another browser) alongside the agent.

> **Note:** Without a browser, the agent shows "default web browser not found" on a fully managed device.

### Fleet UI

1. In Fleet, head to the **Software** page and select your fleet in the fleets dropdown.
2. Select **Add software > App store**, choose the **Android** platform, enter `com.android.chrome` as the application ID, and select **Add software**.
3. Repeat step 2 with `com.cloudflare.cloudflareoneagent` for the agent. The application ID is the app's Google Play package name.
4. On the agent's **Software details** page, select **Actions > Edit configuration**.
5. Paste the JSON from [Write the managed configuration](#write-the-managed-configuration) and select **Save**.
6. To install both apps during setup on newly enrolled devices, head to **Controls > Setup experience > Install software**, select the **Android** tab, select **Add software**, choose Chrome and the agent, and select **Save**.

> **Note:** Setup experience software only applies to devices that enroll afterward. To get the apps on an already enrolled device, install them from the host's **Software** tab or re-enroll the device.

### GitOps

1. Save the managed configuration in your repo, for example at `platforms/android/managed-app-configurations/cloudflare-one-agent.json`.
2. In your fleet's YAML file (for example, `fleets/<fleet>.yml`), add both apps:

   ```yaml
   software:
     app_store_apps:
     - app_store_id: com.cloudflare.cloudflareoneagent
       platform: android
       self_service: true
       setup_experience: true
       configuration:
         path: ../platforms/android/managed-app-configurations/cloudflare-one-agent.json
     - app_store_id: com.android.chrome
       platform: android
       self_service: true
       setup_experience: true
   ```

3. Merge the change.

The `app_store_id` for an Android app is its Google Play package name.

> **Note:** `setup_experience: true` only applies to devices that enroll after you merge the change. To get the apps on an already enrolled device, install them from the host's **Software** tab or re-enroll the device.

> **Note:** GitOps doesn't expand `${VAR}` environment variables in the file you reference in `configuration.path`, so they reach the device as literal text.

> **Note:** GitOps expands environment variables in the whole YAML file, comments included, and rejects `FLEET_SECRET_*` anywhere except profiles and scripts. A comment in your YAML that names a secret variable fails the dry run.

## Turn on always-on VPN

Skip this section if you don't need the VPN on at all times.

Always-on VPN uses an Android configuration profile. Start with lockdown off. Save this as `always-on-vpn-cloudflare.json`:

```json
{
  "alwaysOnVpnPackage": {
    "packageName": "com.cloudflare.cloudflareoneagent",
    "lockdownEnabled": false
  }
}
```

> **Warning:** With `lockdownEnabled: true`, the device has no network access unless the VPN is connected. If the agent can't connect, the device can't receive a profile removal and needs a factory reset. Always roll out always-on VPN without lockdown first.

### Fleet UI

1. Head to **Controls > OS settings > Configuration profiles** and select your fleet in the fleets dropdown.
2. Select **Add profile** and upload `always-on-vpn-cloudflare.json`.
3. [Verify](#verify) that the agent stays connected.
4. Change `lockdownEnabled` to `true` in your file. Then edit the profile in Fleet and upload the updated file as a replacement.

### GitOps

1. Save the profile in your repo, for example at `platforms/android/configuration-profiles/always-on-vpn-cloudflare.json`.
2. Reference the profile in your fleet's YAML file:

   ```yaml
   controls:
     android_settings:
       configuration_profiles:
       - path: ../platforms/android/configuration-profiles/always-on-vpn-cloudflare.json
   ```

3. Merge the change and [verify](#verify) that the agent stays connected.
4. Change `lockdownEnabled` to `true` and merge again.

## Verify

1. On the device, open the Cloudflare One Agent and sign in with a one-time PIN. The device registers to your team.
2. In Chrome, which is in `tunneled_apps`, go to `https://www.cloudflare.com/cdn-cgi/trace`. The page shows `warp=plus`.
3. In a browser that isn't in `tunneled_apps`, such as Firefox, go to the same page. The page shows `warp=off`. This control proves that only the listed apps are tunneled, not the whole device.
4. If you turned on always-on VPN, confirm the agent shows as always-on. It stays connected through airplane mode and a reboot.
5. If you have a Gateway DNS policy that blocks a test domain, turn off Chrome's **Secure DNS**, then go to the domain. The block page appears and the trace page shows `gateway=on`.

To change which apps are tunneled, edit `tunneled_apps`. In the Fleet UI, select **Actions > Edit configuration** on the agent's **Software details** page. With GitOps, edit the JSON file and merge the change. Fleet applies the new configuration without reinstalling the agent. The app's install status shows **Pending** until the configuration applies.

## Troubleshoot

**The agent shows "default web browser not found"**

Install a browser through Fleet, as described in [Add the agent and a browser](#add-the-agent-and-a-browser).

**`https://one.one.one.one/help` says "Using DNS over WARP: No" on a tunneled browser**

This page is misleading on Android. Chrome's Secure DNS and Android's Private DNS talk to 1.1.1.1 directly. Use `https://www.cloudflare.com/cdn-cgi/trace` instead. It shows `warp=plus` when the device is enrolled in Zero Trust and `gateway=on` when Gateway applies.

**A Gateway DNS policy doesn't block a domain**

The browser's own DNS over HTTPS (DoH) can bypass Gateway. Turn off Chrome's **Secure DNS** and check Android's **Private DNS** setting. Firefox has its own DoH setting too.

**The agent shows "authentication required" or "incorrect credentials"**

You're likely using service-token enrollment with `$FLEET_SECRET_*`. The agent reads `auth_client_id` and `auth_client_secret` from the managed configuration, but Fleet doesn't substitute secret variables there. See the warning in [Prerequisites](#prerequisites).

**The device has no network after enabling lockdown**

The agent isn't connected, and lockdown blocks all traffic without it. Factory reset the device. To avoid this, enable lockdown only after you confirm the agent connects with lockdown off.

## Further reading

- [Cloudflare One Client MDM deployment parameters](https://developers.cloudflare.com/cloudflare-one/team-and-resources/devices/cloudflare-one-client/deployment/mdm-deployment/parameters/)
- [Install app store apps](https://fleetdm.com/guides/install-app-store-apps)
- [GitOps YAML reference: app_store_apps](https://fleetdm.com/docs/configuration/yaml-files#app_store_apps)

<meta name="articleTitle" value="Deploy Cloudflare One Agent on Android">
<meta name="authorFullName" value="Kitzy">
<meta name="authorGitHubUsername" value="kitzy">
<meta name="publishedOn" value="2026-10-13">
<meta name="category" value="guides">
<meta name="description" value="Deploy Cloudflare One Agent to fully managed Android devices with Fleet, including per-app VPN, always-on VPN, and Gateway DNS filtering.">
