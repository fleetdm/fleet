# Fleet 4.31.0 | MDM enrollment workflow, API user role.

![Fleet 4.31.0](../website/assets/images/articles/fleet-4.31.0-1600x900@2x.png)

Fleet 4.31.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.31.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

* MDM enrollment workflow
* mTLS support in Fleet
* GitOps user role

### MDM enrollment workflow

Hot on the heels of Fleet’s [public beta announcement](https://fleetdm.com/releases/fleet-introduces-mdm) for MDM, we continue to provide 🟢 Results by adding several new features in the enrollment workflow, including SAML authentication and complete support for bootstrap packages.

Authentication for device enrollments enables you to integrate the MDM enrollment process with an identity provider. With Fleet, you can configure authentication for enrollments using any identity provider service that supports custom SAML integrations, including Google Workspace, Microsoft Azure, Okta, OneLogin, and JumpCloud, to name a few. Once set up, users must authenticate against their identity provider using their credentials to successfully enroll their devices into Fleet. 

In addition to Fleet Desktop, Fleet can also deliver a bootstrap software package or installer that to initiate the deployment and configuration of devices during the enrollment process. Use a bootstrap package to deploy your own configuration manager, such as Munki, Chef, or Puppet. With a seamless deployment and configuration of your preferred configuration manager on devices enrolled in Fleet, you will have more granular control and customized management of your organization's devices.

### mTLS support in fleetd (Orbit)

_Available in Fleet Premium and Fleet Ultimate_

Mutual TLS (mTLS) ensures secure and authenticated communication between two parties. Unlike traditional TLS, where only the server's identity is verified, mTLS requires both the server and the client to authenticate each other using digital certificates. This additional layer of security helps prevent unauthorized access and enhances data privacy. mTLS is often leveraged in a zero-trust networking environment, because security measures are applied regardless of whether the user or device is inside or outside the network perimeter.

Fleet is bringing 🟢 Results with support for TLS client certificates in fleetd (Orbit) to ensure secure communication to fleet. Learn more about [using mTLS certificates](https://fleetdm.com/docs/using-fleet/orbit#orbit-mtls-support) when generating your Fleet packages.

### GitOps user role

_Available in Fleet Premium and Fleet Ultimate_

Take 🟠 Ownership of Fleet account roles with greater granularity. Fleet 4.31.0 includes a new user role, `gitops`. 

The `gitops` user role is ideal for automated workflows as part of continuous integration/continuous development (CI/CD) actions, such as MDM profile commitments and security profiles. The `gitops` user role can only access Fleet using the API and is unable to access the Fleet dashboard.

## More new features, improvements, and bug fixes

#### List of features
    
* Added `gitops` user role to Fleet. GitOps users are users that can manage configuration.
* Added the `fleetctl get mdm-commands` command to get a list of MDM commands that were executed. Added the `GET /api/latest/fleet/mdm/apple/commands` API endpoint.
* Added Fleet UI flows for uploading, downloading, deleting, and viewing information about a Fleet MDM bootstrap package.
* Added `apple_bm_enabled_and_configured` to app config responses.
* Added support for the `mdm.macos_setup.macos_setup_assistant` key in the 'config' and 'team' YAML payloads supported by `fleetctl apply`.
* Added the endpoints to set, get and delete the macOS setup assistant associated with a team or no team (`GET`, `POST` and `DELETE` methods on the `/api/latest/fleet/mdm/apple/enrollment_profile` path).
* Added functionality to gate Apple MDM login behind SAML authentication.
* Added new "verifying" status for MDM profiles.
* Migrated MDM status values from "applied" to "verifying" and updated associated endpoints.
* Updated macOS settings status filters and aggregate counts to more accurately reflect the status of FileVault settings.
* Filter out non-`observer_can_run` queries for observers in `fleetctl get queries` to match the UI behavior.
* Fall back to a previous NVD release if the asset we want is not in the latest release.
* Users can now click back to software to return to the filtered host details software tab or filtered manage software page.
* Users can now bookmark software table filters.
* Added a maximum height to the teams dropdown, allowing the user to scroll through a large number of teams.
* Present the 403 error page when a user with no access logs in.
* Back to hosts and back to software in host details and software details return to previous table state.
* Bookmarkable URLs are now the source of truth for Manage Host and Manage Software table states.
* Removed old Okta configuration that was only documented for internal usage. These configs are being replaced for a general approach to gate profiles behind SSO.
* Removed any host's packs information for observers and observer plus in UI.
* Added `changed_macos_setup_assistant` and `deleted_macos_setup_assistant` activities for the macOS setup assistant setting.
* Hide reset sessions in user dropdown for current user.
* Added a suite of UI logic for premium features in the Sandbox environment.
* In Sandbox, added "Premium Feature" icons for premium-only option to designate a policy as "Critical," as well as copy to the tooltip above the icon next to policies designated "Critical" in the Manage policies table.
* Added a star to let a sandbox user know that the "Probability of exploit" column of the Manage Software page is a premium feature.
* Added "Premium Feature" icons for premium-only columns of the Vulnerabilities table when in Sandbox mode.
* Inform prospective customers that Teams is a Premium feature.
* Fixed animation for opening edit user modal.
* Fixed nav bar buttons not responsively resizing when small screen widths cannot fit default size nav bar.
* Fixed a bug with and improved the overall experience of tabbed navigation through the setup flow.
* Fixed `/api/_version/fleet/logout` to return HTTP 401 if unauthorized.
* Fixed endpoint to return proper status code (401) on `/api/fleet/orbit/enroll` if secret is invalid.
* Fixed a bug where a white bar appears at the top of the login page before the app renders.
* Fixed bug in manage hosts table where UI elements related to row selection were displayed to a team observer user when that user was also a team and maintainer or admin on another team.
* Fixed bug in add policy UI where a user that is team maintainer or team admin cannot access the UI to save a new policy if that user is also an observer on another team.
* Fixed UI bug where dashboard links to hosts filtered by platform did not carry over the selected team filter.
* Fixed not showing software card on dashboard when clicking on vulnerabilities.
* Fixed a UI bug where fields on the "My account" page were cut off at smaller viewport widths.
* Fixed software table to match UI spec (responsively hidden vulnerabilities/probability of export column under 990px width).
* Fixed a bug where bundle information displayed in tooltips over a software's name was mistakenly hidden.
* Fixed an HTTP 500 on `GET /api/_version_/fleet/hosts` returned when `mdm_enrollment_status` is invalid.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.31.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2023-05-01">
<meta name="articleTitle" value="Fleet 4.31.0 | MDM enrollment workflow, API user role.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.31.0-1600x900@2x.png">
