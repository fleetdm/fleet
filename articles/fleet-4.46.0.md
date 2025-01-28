# Fleet 4.46.0 | Automatic SCEP certificate renewal.

![Fleet 4.46.0](../website/assets/images/articles/fleet-4.46.0-1600x900@2x.png)

Fleet 4.46.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.45.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

* Automatic SCEP certificate renewal


### Automatic SCEP certificate renewal

Fleet is implementing an automatic renewal process for Simple Certificate Enrollment Protocol (SCEP) certificates on hosts. This ensures that SCEP certificates are renewed within 30 days of the certificate expiration date, eliminating the need for manual intervention or reactivation of MDM for macOS hosts. Fleet proactively maintains the validity of SCEP certificates, enhancing security and operational efficiency. This update aligns with Fleet's commitment to reliability and proactive management, ensuring that IT administrators can maintain continuous security compliance without the administrative burden of manually monitoring and renewing certificates.




## Changes

* Fixed issues with how errors were captured in Sentry:
        - The stack trace is now more precise.
        - More error paths were captured in Sentry.
        - **Note: Many more entries could be generated in Sentry compared to earlier Fleet versions. Sentry capacity should be planned accordingly.**
- User settings/profile page officially renamed to account page
- UI Edit team more properly labeled as rename team
- Fixed issue where the "Type" column was empty for Windows MDM profile commands when running `fleetctl get mdm-commands` and `fleetctl get mdm-command-results`.
- Upgraded Golang version to 1.21.7
- Updated UI's empty policy states
* Automatically renewed macOS identity certificates for devices 30 days prior to their expiration.
* Fixed bug where updating policy name could result in multiple policies with the same name in a team.
  - This bug was introduced in Fleet v4.44.1. Any duplicate policy names in the same team were renamed by adding a number to the end of the policy name.
- Fixed an issue where some MDM profile installation errors would not be shown in Fleet.
- Deleting a policy updated the policy count
- Moved show query button to show in report page even with no results
- Updated page description styling
- Fixed UI loading state for software versions and OS for the initial request.

## Fleet 4.45.1 (Feb 23, 2024)

### Bug fixes

* Fixed a bug that caused macOS ADE enrollments gated behind SSO to get a "method not allowed" error.
* Fixed a bug where the "Done" button on the add hosts modal for plain osquery could be covered.


## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.46.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2024-02-26">
<meta name="articleTitle" value="Fleet 4.46.0 | Automatic SCEP certificate renewal.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.46.0-1600x900@2x.png">
