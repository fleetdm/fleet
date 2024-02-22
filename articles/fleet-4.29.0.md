# Fleet 4.29.0 | SSO provides JIT Fleet user roles.

![Fleet 4.29.0](../website/assets/images/articles/fleet-4.29.0-1600x900@2x.png)

Fleet 4.29.0 is up and running. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.29.0) or continue reading to get the highlights.

For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

*   SSO provides JIT Fleet user roles
*   CIS benchmarks manual intervention
*   Critical policies


## SSO provides JIT Fleet user roles

_Available in Fleet Premium and Fleet Ultimate_

<div purpose="embedded-content">
    <iframe src="https://www.youtube.com/embed/qGe1bgYzxvg" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen></iframe>
</div>

With this update, you can take 🟠 Ownership of Fleet account roles assignment when using  Just-in-time (JIT) provisioning. When JIT user provisioning is enabled, Fleet automatically creates a user account upon first login with the configured single sign-on (SSO). The email and full name are copied from the user data in the SSO during the creation process. Large organizations no longer need to create individual users. By default, accounts created via JIT provisioning are assigned the [Global Observer role](https://fleetdm.com/docs/using-fleet/permissions).

Users created via JIT provisioning can be assigned Fleet roles using SAML custom attributes sent by the IdP in a `SAMLResponse` during login. Global or team roles can be assigned one of the supported values: admin, maintainer, and observer. Fleet will attempt to parse SAML custom attributes. If the account exists, and `enable_jit_role_sync` is true, the Fleet account roles will be updated to match those set in the SAML custom attributes at every login.

Learn more about [JIT user role setting](https://fleetdm.com/docs/deploying/configuration#just-in-time-jit-user-provisioning).

## CIS benchmarks manual intervention

_Available in Fleet Premium and Fleet Ultimate_

<div purpose="embedded-content">
    <iframe src="https://www.youtube.com/embed/9h38yEIuE6c" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share" allowfullscreen></iframe>
</div>

The Center for Internet Security (CIS) publishes benchmark documents describing the proper configuration of computers to avoid vulnerabilities addressed therein. Fleet 4.28 included scheduling and running a complete set of [CIS benchmark policies](https://fleetdm.com/docs/using-fleet/cis-benchmarks) as part of Premium and Ultimate. Today, Fleet has added additional macOS 13 Ventura CIS benchmarks that can be detected but require manual intervention.

CIS benchmark policies represent the consensus-based effort of cybersecurity experts globally to help protect your systems against threats more confidently. Fleet takes 🟠 Ownership toward providing the most comprehensive CIS benchmark policies available. Using Fleet to detect these additional CIS policies will assist you in quickly bringing your fleet into compliance, saving your organization time and money.

Learn more about [macOS 13.0 Ventura Benchmark manual checks](https://fleetdm.com/docs/using-fleet/cis-benchmarks#mac-os-13-0-ventura-benchmark-manual-checks-that-require-customer-decision).

### Vulnerability management improvement

Fleet updated translation rules to provide better 🟢 Results and avoid false positives when reporting on the Docker desktop. With these changes, the Docker desktop is now mapped to the proper CVE, fixing the false positive where the Docker desktop was showing vulnerabilities that should have been associated with the Docker engine.

## More new features, improvements, and bug fixes

#### List of MDM features

* Added activity feed items for enabling and disabling disk encryption with MDM.
* Added FileVault banners on the Host Details and My Device pages.
* Added activities for when macOS disk encryption setting is enabled or disabled.
* Added UI for Fleet MDM managed disk encryption toggling and the disk encryption aggregate data.
* Added support to update a team's disk encryption via the Modify Team (`PATCH /api/latest/fleet/teams/{id}`) endpoint.
* Added a new API endpoint to gate access to an enrollment profile behind Okta authentication.
* Added new configuration values to integrate Okta in the DEP MDM flow.
* Added `GET /mdm/apple/profiles/summary` endpoint.
* Updated API endpoints that use `team_id` query parameter so that `team_id=0 \
`filters results to include only hosts that are not assigned to any team.
* Adjusted the `aggregated_stats` table to compute and store statistics for "no team" in addition to per-team and for all teams.
* Added MDM profiles status filter to hosts endpoints.
* Added indicators of aggregate host count for each possible status of MDM-enforced mac settings (hidden until 4.30.0).

#### List of other features

* As part of JIT provisioning, read user roles from SAML custom attributes.
* Added Win 10 policies for CIS Benchmark 18.x.
* Added Win 10 policies for CIS Benchmark 2.3.17.x.
* Added Win 10 policies for CIS Benchmark 2.3.10.x.
* Documented CIS Windows10 Benchmarks 9.2.x to cis policy queries.
* Document CIS Windows10 Benchmarks 9.3.x to cis policy queries.
* Added button to show query on policy results page.
* Run periodic cleanup of pending `cron_stats` outside the `schedule` package to prevent Fleet outages from breaking cron jobs.
* Added an invitation for users to upgrade to Premium when viewing the Premium-only "macOS updates" feature.
* Added an icon on the policy table to indicate if a policy is marked critical.
* Added `"instanceID"` (aka `owner` of `locks`) to `schedule` logging (to help troubleshoot when running multiple Fleet instances).
* Introduce UUIDs to Fleet errors and logs.
* Added EndeavourOS, Manjaro, openSUSE Leap, and Tumbleweed to HostLinuxOSs.
* Global observer can view settings for all teams.
* Team observers can view the team's settings.
* Updated translation rules so that Docker Desktop can be mapped to the correct CPE.
* Pinned Docker image hashes in Dockerfiles for increased security.
* Remove the `ATTACH` check on SQL osquery queries (osquery bug fixed a while ago in 4.6.0).
* Don't return internal error information on Fleet API requests (internal errors are logged to stderr).
* Fixed an issue when applying the configuration YAML returned by `fleetctl get config` with \
`fleetctl apply` when MDM is not enabled.
* Fixed a bug where `fleetctl trigger` doesn't release the schedule lock when the triggered run spans the regularly scheduled interval.
* Fixed a bug that prevented starting the Fleet server with MDM features if Apple Business Manager (ABM) was not configured.
* Fixed incorrect MDM-related settings documentation and payload response examples.
* Fixed bug to keep team when clicking on policy tab twice.
* Fixed software table links that were cutting off tooltips.

## Ready to upgrade?

Visit our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.29.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2023-03-22">
<meta name="articleTitle" value="Fleet 4.29.0 | SSO provides JIT Fleet user roles">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.29.0-1600x900@2x.png">
