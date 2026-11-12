# Fleet 4.95.0 | Okta conditional access for Windows, built-in MCP server, AI tools in inventory, and more...

<!--
<div purpose="embedded-content">
   <iframe src="https://www.youtube.com/embed/TODO" allowfullscreen></iframe>
</div>
-->

Fleet 4.95.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.95.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- [Okta conditional access for Windows](#okta-conditional-access-for-windows)
- [Google conditional access for iOS/iPadOS, and Android](#google-conditional-access-for-ios-ipados-and-android)
- [Built-in MCP server](#built-in-mcp-server)
- [AI tools in software inventory](#ai-tools-in-software-inventory)
- [Patch deadline for macOS apps](#patch-deadline-for-macos-apps)
- [Multiple versions of the same App Store app](#multiple-versions-of-the-same-app-store-app)
- [Multiple VPP tokens in one fleet](#multiple-vpp-tokens-in-one-fleet)
- [Deploy certificates from EJBCA and Google CAS](#deploy-certificates-from-ejbca-and-google-cas)

### Okta conditional access for Windows

IT admins can now use Okta conditional access on Windows hosts. End users on hosts that fail a conditional access policy are blocked from signing in to Okta apps until they fix the issue.

Learn how in the [Okta conditional access guide](https://fleetdm.com/guides/okta-conditional-access-integration#windows).

GitHub issue: [#53284](https://github.com/fleetdm/fleet/issues/53284)

### Google conditional access for iOS/iPadOS, and Android

IT admins whose organization uses Google Workspace can now block sign-in from iPhones, iPads, and Android devices that aren't company-owned and enrolled to Fleet. Fleet installs Chrome with a bookmark on each company-owned device. When the end user opens it, Fleet confirms the device is enrolled and company-owned, and the device is marked as managed in Google. Google's [Context-Aware Access](https://knowledge.workspace.google.com/admin/security/about-context-aware-access) then blocks devices that aren't.

Learn how in the [Google conditional access guide](https://fleetdm.com/guides/google-conditional-access-integration).

GitHub issue: [#54888](https://github.com/fleetdm/fleet/issues/54888)

### Built-in MCP server

Fleet now includes a built-in MCP server, so AI tools like Claude, Codex, and Copilot can look up hosts, run live reports, check policies, and investigate CVEs in Fleet. There's nothing extra to deploy. Connect your AI tool to `https://<your-fleet-url>/api/v1/fleet/mcp` with an API-only user's token. The AI tool only gets the tools that the API-only user's role and API endpoints allow.

Learn how in the [guide](https://fleetdm.com/guides/connect-your-ai-tool-to-fleet)

GitHub issue: [#44448](https://github.com/fleetdm/fleet/issues/44448)

### AI tools in software inventory

Fleet now shows the AI tools installed on all your hosts. On the **Software** page, **Host details**, and **My device**, open **Filters** and turn on **AI tools**. Fleet adds three software types for AI tools it didn't collect before: **MCP server**, **AI skill**, and **AI CLI tool**. AI apps and extensions that Fleet already collects, like the ChatGPT app or the GitHub Copilot VS Code extension, keep their type and are marked as AI tools. In the API, use the `ai_tool` filter on [list software](https://fleetdm.com/docs/rest-api/rest-api#list-software).

See what Fleet collects for each kind of AI tool in the [software inventory reference](https://fleetdm.com/guides/software-inventory-reference#ai-tools).

GitHub issue: [#51599](https://github.com/fleetdm/fleet/issues/51599)

### Patch deadline for macOS apps

IT admins can now patch macOS Fleet-maintained apps on a schedule that matches their organization's patching rules (SLAs). Choose **Force patch: Deadline** on a patch policy, then set the deadline's days and time for each fleet in **Software > Library > Settings**. Deadlines use each host's local time. The default is Tuesday at 5pm end user local time.

If the app is closed when the policy fails, Fleet patches it right away. If it's open, Fleet Desktop notifies the end user 1 hour before the deadline and again 5 minutes before, then closes and updates the app. Apps waiting on the same deadline show up together in one notification.

Learn more in the [patching end user experience guide](https://fleetdm.com/guides/patching-end-user-experience#force-patch-deadline).

GitHub issue: [#39176](https://github.com/fleetdm/fleet/issues/39176)

### Multiple versions of the same App Store app

IT admins can now add more than one version of the same App Store app to a fleet for iOS, iPadOS, and Android. Each version has its own app configuration, automatic update window (iOS/iPadOS only), self-service setting, categories, and labels. For example, scope one Zoom version with a VPN or SSO configuration to your "Production" label, and another to your "Test" label. If a host is in the scope of more than one version, Fleet installs the version that was added to Fleet first.

Learn more in the [App Store apps guide](https://fleetdm.com/guides/install-app-store-apps#multiple-app-instances).

GitHub issue: [#47904](https://github.com/fleetdm/fleet/issues/47904)

### Multiple VPP tokens in one fleet

IT admins can now assign more than one Volume Purchasing Program (VPP) token to a fleet. This helps international organizations that keep all their iPhones and iPads in one fleet. For example, assign both a US and a Japan token to the fleet, and add Japan-only App Store apps from the Japan token.

When you add an App Store app, choose which token it comes from. Each app installs with the token it was added from. In the API, pass `vpp_token_id` to [add App Store app](https://fleetdm.com/docs/rest-api/rest-api#add-app-store-app). In GitOps, use [`location`](https://fleetdm.com/docs/configuration/yaml-files#multiple-vpp-tokens) to pick the token.

Learn more in the [Apple MDM setup guide](https://fleetdm.com/guides/apple-mdm-setup#volume-purchasing-program-vpp).

GitHub issue: [#53835](https://github.com/fleetdm/fleet/issues/53835)

### Deploy certificates from EJBCA and Google CAS

IT admins who use [EJBCA](https://www.ejbca.org/) as their certificate authority (CA) can now deploy certificates to macOS, iOS, and iPadOS hosts, so end users can connect to corporate Wi-Fi or VPN. Add EJBCA in **Settings > Integrations > Certificate authorities**, with the API, or in [GitOps](https://fleetdm.com/docs/configuration/yaml-files#ejbca). Then use the `$FLEET_VAR_EJBCA_DATA_<name>` and `$FLEET_VAR_EJBCA_PASSWORD_<name>` variables in a PKCS#12 configuration profile.

Learn how in the [certificates guide](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#ejbca).

IT admins who use Google Cloud's [Certificate Authority Service (CAS)](https://cloud.google.com/certificate-authority-service) can now deploy certificates to Linux hosts. Connect Fleet to Google CAS, then deploy a script-only package that requests a certificate with Fleet's [request certificate API](https://fleetdm.com/docs/rest-api/rest-api#request-certificate). A policy runs the same script to renew the certificate before it expires.

Learn how in the [certificates guide](https://fleetdm.com/guides/connect-end-user-to-wifi-with-certificate#google-cas).

GitHub issues: [#30986](https://github.com/fleetdm/fleet/issues/30986) and [#52623](https://github.com/fleetdm/fleet/issues/52623)

## Changes

- Removed fleetd's built-in Nudge integration. Fleet no longer sends Nudge settings to fleetd for hosts on macOS 13 and older, which are no longer supported. Hosts on macOS 14 and later continue to get OS update enforcement via DDM. To keep using Nudge, deploy it with the Nudge Fleet-maintained app. Learn how in the [Nudge guide](https://fleetdm.com/guides/enforce-os-updates-on-macos-with-nudge).

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.95.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Luke Heath">
<meta name="authorGitHubUsername" value="lukeheath">
<meta name="publishedOn" value="2026-11-16">
<meta name="articleTitle" value="Fleet 4.95.0 | Okta conditional access for Windows, built-in MCP server, AI tools in inventory, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.95.0-1600x900@2x.png">
