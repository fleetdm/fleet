# Fleet 4.85.0 | Vulnerability exposure dashboard, local admin accounts, dark mode, and more...

Fleet 4.85.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.85.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- [Vulnerability exposure dashboard](#vulnerability-exposure-dashboard)
- [More accurate vulnerability data](#more-accurate-vulnerability-data)
- [Pin Fleet-maintained apps to a major version](#pin-fleet-maintained-apps-to-a-major-version)
- [Create a local admin account during macOS setup](#create-a-local-admin-account-during-macos-setup)
- [Scoped API-only users](#scoped-api-only-users)
- [Dark mode](#dark-mode)


### Vulnerability exposure dashboard

Fleet now includes a vulnerability exposure report that tracks your organization's [patching progress](https://fleetdm.com/articles/how-to-use-policies-for-patch-management-in-fleet) over time. The chart covers critical vulnerabilities in major browsers, Microsoft Office, operating systems, and Adobe Reader. The report joins Fleet's growing dashboard alongside new "Hosts online" and "Hosts enrolled" reports also added in 4.85.

GitHub issue: [#43769](https://github.com/fleetdm/fleet/issues/43769)

### More accurate vulnerability data

Fleet has migrated Red Hat Enterprise Linux (RHEL) 8 and 9 [vulnerability (CVE) scanning](https://fleetdm.com/articles/vulnerability-processing) from OVAL XML feeds to OSV JSON. This is the format Red Hat began publishing natively in November 2024. This eliminates a class of false positives: OVAL grouped CVEs by advisory and sometimes attributed them to packages that weren't actually vulnerable, while OSV maps each CVE to exact affected package versions. No Fleet configuration changes are required; the transition happens automatically on upgrade.

GitHub issue: [#40056](https://github.com/fleetdm/fleet/issues/40056)

### Pin Fleet-maintained apps to a major version

IT admins using [GitOps](https://fleetdm.com/docs/configuration/yaml-files) can now pin a [Fleet-maintained app](https://fleetdm.com/software-catalog) to a specific major version using a caret constraint (e.g. `^3`). Hosts stay patched because Fleet automatically installs updates within that major version but won't install a new major release you haven't tested or licensed. Set it once in your YAML and patching takes care of itself within the version you control. Versioning pinning in the UI is [coming soon](https://github.com/fleetdm/fleet/issues/38504).

GitHub issue: [#38988](https://github.com/fleetdm/fleet/issues/38988)

### Create a local admin account during macOS setup

During macOS [Automated Device Enrollment (ADE)](https://fleetdm.com/articles/apple-device-enrollment-program), Fleet can now create a hidden admin account. This gives IT admins a way in if hands-on access is otherwise needed. Admins can view and copy the generated password, unique per-host, from the **Host details** page. Activity is logged on account creation and password views.

GitHub issue: [#37141](https://github.com/fleetdm/fleet/issues/37141)

### Scoped API-only users

Fleet Premium now supports scoped API-only users, letting us restrict a token to a specified list of allowed API endpoints. If a token leaks, the blast radius is limited to those endpoints. Scoped API-only users can be created via the Fleet UI, [`fleetctl`](https://fleetdm.com/articles/fleetctl), or the [REST API](https://fleetdm.com/docs/rest-api/rest-api).

GitHub issue: [#38044](https://github.com/fleetdm/fleet/issues/38044)


### Dark mode

Fleet now ships with a dark theme. Now, by default, Fleet automatically follows your OS light/dark mode preference. If you want to choose, you can pick between modes on your **My account** page. Whether you're working in the dark or just prefer dark mode on principle, Fleet now looks the part.

GitHub issue: [#42977](https://github.com/fleetdm/fleet/issues/42977)

## Changes

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.85.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-05-13">
<meta name="articleTitle" value="Fleet 4.85.0 | Vulnerability exposure dashboard, local admin accounts, dark mode, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.85.0-1600x900@2x.png">
