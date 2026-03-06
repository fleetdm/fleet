# Fleet 4.82.0 | Fleets and reports, new technician role, and more...

<div purpose="embedded-content">
   <iframe src="TODO" title="0" allowfullscreen></iframe>
</div>

Fleet 4.82.0 is now available. See the complete [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.82.0) or read on for highlights. For upgrade instructions, visit the [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

- Fleets and reports rename
- New technician role for helpdesk teams
- Self-service scripts on macOS

### Fleets and reports rename

Fleet now uses "fleets" instead of "teams" and "reports" instead of "queries" across the UI, API, CLI, and GitOps (YAML). The new "fleets" terminology better reflects how hosts are grouped and managed in Fleet. "Reports" makes it clearer that these are used to collect host information.

Existing workflows continue to work. All API endpoints, CLI commands, and YAML Keys with "teams" and "queries" are still supported for backward compatibility and automatically map to "fleets" and "reports" respectively. Reference documentation updates are coming soon.

GitHub issues: [#39314](https://github.com/fleetdm/fleet/issues/39314), [#39238](https://github.com/fleetdm/fleet/issues/39238) 

### New technician role

Fleet now includes a Technician role designed for helpdesk and IT support teams. Technicians can run scripts, view results, and install or uninstall software. Check out the [permissions table](https://fleetdm.com/guides/role-based-access#user-permissions) for a full list of permissions.

This enables least-privilege access for day-to-day support tasks while keeping sensitive configuration settings restricted.

GitHub issue: [#35696](https://github.com/fleetdm/fleet/issues/35696)

### Self-service scripts on macOS

Fleet now supports self-service scripts through script-only packages on macOS. Upload a `.sh` script and make it available in self-service for end users on macOS hosts. [Learn how]().

This makes it easier for IT admins to deliver quick fixes and utility scripts.

GitHub issue: [#33951](https://github.com/fleetdm/fleet/issues/33951)

### Manage fully-managed Android hosts

Fleet now supports managing company-owned Android hosts in fully-managed mode. This allows IT teams to apply stricter controls and use Android management features that aren’t available on BYOD Android hosts (work profiles). Learn how to [enroll Android hosts](https://fleetdm.com/guides/enroll-hosts).

GitHub issue: [#36337](https://github.com/fleetdm/fleet/issues/36337)

## Changes

TODO

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs to update to Fleet 4.81.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="Noah Talerman">
<meta name="authorGitHubUsername" value="noahtalerman">
<meta name="publishedOn" value="2026-03-05">
<meta name="articleTitle" value="Fleet 4.82.0 | Fleets and reports, new technician role, and more...">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.82.0-1600x900@2x.png">
