# Fleet 4.89.0 | Windows setup experience improvements, Android variables everywhere, and more...

<div purpose="embedded-content">
   <iframe src="TODO" title="0" allowfullscreen></iframe>
</div>

Fleet 4.89.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.89.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- [Windows setup experience: continue past a failed install](#windows-setup-experience-continue-past-a-failed-install)
- [Android: host vital variables everywhere](#android-host-vital-variables-everywhere)
- [Default fleet for BYOD iOS/iPadOS enrollment](#default-fleet-for-byod-ios-ipados-enrollment)
- [Auto-update, pin, and roll back Fleet-maintained apps](#auto-update-pin-and-roll-back-fleet-maintained-apps)
- [Filter and save the vulnerability exposure chart](#filter-and-save-the-vulnerability-exposure-chart)
- [Policy status page](#policy-status-page)
- [Script-only packages: pre-install query, post-install, and uninstall scripts](#script-only-packages-pre-install-query-post-install-and-uninstall-scripts)
- [IdP host vitals from Google Workspace](#idp-host-vitals-from-google-workspace)

### Windows setup experience: continue past a failed install

_Available in Fleet Premium_

When required setup software fails to install during [Windows automatic enrollment](https://fleetdm.com/guides/windows-mdm-setup#automatic-enrollment) (Autopilot or non-Autopilot), end users now see exactly which software failed. If the IT admin hasn't checked **Cancel setup if software fails**, the end user can continue past the failure and install the missing software later from self-service. If that option is checked, setup stops and the end user is told to reset the device and try again. Either way, end users get a clear next step instead of a stuck setup screen, which means fewer support tickets for IT admins.

GitHub issue: [#45948](https://github.com/fleetdm/fleet/issues/45948)

### Android: host vital variables everywhere

IT admins can now use any host vital variable (`$FLEET_VAR_HOST_`), like a host's UUID or the end user's IdP email, in Android configuration profiles, certificate templates, and managed app configuration. This makes it possible to deploy a host-specific value as part of an app's configuration, for example, passing a host's UUID to Duo as a trusted endpoint identifier, or a user's email as the identity for [EAP-TLS Wi-Fi authentication](https://fleetdm.com/guides/configure-eap-tls-wifi-android). For certificates, Fleet also detects when a host vital variable's value changes and automatically resends the certificate so it stays accurate. See all host vital variables in the [built-in variables guide](https://fleetdm.com/guides/fleet-variables).

GitHub issues: [#45353](https://github.com/fleetdm/fleet/issues/45353), [#41968](https://github.com/fleetdm/fleet/issues/41968), [#37406](https://github.com/fleetdm/fleet/issues/37406)

### Default fleet for BYOD iOS/iPadOS enrollment

IT admins can now choose a default fleet for iOS and iPadOS hosts that [enroll via Account-driven User Enrollment (BYOD)](https://fleetdm.com/guides/enroll-personal-byod-ios-ipad-hosts-with-managed-apple-account). This means personal iPhones and iPads automatically land in the right fleet on enrollment, so they get the correct configuration profiles and software without an admin having to move them manually.

GitHub issue: [#30871](https://github.com/fleetdm/fleet/issues/30871)

### Auto-update, pin, and roll back Fleet-maintained apps

_Available in Fleet Premium_

IT admins can now control exactly which version of a [Fleet-maintained app](https://fleetdm.com/guides/fleet-maintained-apps) their hosts run. Pin a Fleet-maintained app to a specific version to stop it from auto-updating, or roll back to the previous version if a new release causes problems, all from the software title's page. If you're relying on auto-update, Fleet checks for new versions hourly, so hosts stay current without an IT admin re-adding the app.

GitHub issue: [#38504](https://github.com/fleetdm/fleet/issues/38504)

### Filter and save the vulnerability exposure chart

_Available in Fleet Premium_

Security Engineers can now filter the [vulnerability exposure chart](https://fleetdm.com/guides/dashboard-vulnerability-exposure) by software category (operating system, browsers, Microsoft Office, or Adobe apps), EPSS exploit probability, known active exploits (CISA KEV), and specific CVEs to exclude, so the chart reflects the risk registry they actually track instead of every vulnerability Fleet detects. These default filters can now be set and persisted via [GitOps (YAML)](https://fleetdm.com/docs/configuration/yaml-files#features), so they load automatically the next time the chart opens. Filters changed directly in the Fleet UI aren't saved, whether GitOps mode is on or off.

GitHub issues: [#44746](https://github.com/fleetdm/fleet/issues/44746), [#47327](https://github.com/fleetdm/fleet/issues/47327)

### Policy status page

IT admins get a historical view of [policy automation](https://fleetdm.com/guides/automations#policy-automations) runs: pass/fail status for every host, alongside the output of the software install or script run that the automation triggered. This makes it much faster to troubleshoot a host that keeps failing a policy, since admins no longer have to dig through separate activity logs to piece together what happened.

GitHub issue: [#38670](https://github.com/fleetdm/fleet/issues/38670)

### Script-only packages: pre-install query, post-install, and uninstall scripts

_Available in Fleet Premium_

IT admins can now add a pre-install query, a post-install script, and an uninstall script to [script-only software packages](https://fleetdm.com/guides/deploy-software-packages#script-only-packages), matching the behavior already available for custom packages. This means script-only packages can now offer an uninstall option and the same install verification other packages already have.

GitHub issue: [#42797](https://github.com/fleetdm/fleet/issues/42797)

### IdP host vitals from Google Workspace

_Available in Fleet Premium_

Fleet users who use Google Workspace (GW) as their identity provider (IdP) can now populate [IdP host vitals](https://fleetdm.com/guides/foreign-vitals-map-idp-users-to-hosts) (group, department, username, email, and full name) directly from GW, without building a custom integration. Since Google Workspace doesn't support the [SCIM protocol](https://scim.cloud/), Fleet pulls directory data from Google's API on a schedule. Once connected, IT admins can scope configuration profiles, software, and policies using IdP host vital labels, the same way they would with an Okta or Entra SCIM integration.

GitHub issue: [#42915](https://github.com/fleetdm/fleet/issues/42915)

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.89.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-07-14">
<meta name="articleTitle" value="Fleet 4.89.0 | Windows setup experience improvements, Android variables everywhere, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.89.0-1600x900@2x.png">
