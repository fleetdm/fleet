# Fleet 4.32.0 | User migration, customizing macOS Setup Assistant.

![Fleet 4.32.0](../website/assets/images/articles/fleet-4.32.0-1600x900@2x.png)

Fleet 4.32.0 is live. Check out the full [changelog](https://github.com/fleetdm/fleet/releases/tag/fleet-v4.32.0) or continue reading to get the highlights.
For upgrade instructions, see our [upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs.

## Highlights

* End user migration
* macOS Setup Assistant custom setup panes per team


### End user migration

_Available in Fleet Premium and Fleet Ultimate_

Achieve 🟢 Results when migrating users to Fleet from your previous MDM solution. Fleet's end user migration workflow allows you to take 🟠 Ownership of your user migration, making moving your fleet to Fleet quick and easy. 

With Fleet installed, users will be prompted to migrate their device to Fleet for management. To build confidence with your users, Fleet allows for full customization of your organization name, logo, and contact link displayed in the prompt. Once initiated, a webhook is triggered, which can be used to kick off a workflow to unenroll the device from the previous MDM solution using Tines, Okta Workflows, Make, and other no-code automation platforms.

Learn more about Fleet's end-user [migration workflow](https://fleetdm.com/docs/using-fleet/mdm-migration-guide).


### macOS Setup Assistant custom setup panes per team

_Available in Fleet Premium and Fleet Ultimate_

Continuing Fleet’s 🟣 Openness and GitOps forward approach to MDM, we are excited to allow full control of the macOS Setup Assistant using an automatic enrollment profile. An automatic enrollment profile allows administrators to select which screens users see in the macOS Setup Assistant using a JSON file. 

With this addition, administrators can have different setup experiences for each team—allowing a conference room computer to skip all setup screens while users see the screens they need. Controlling the macOS Setup Assistant using a JSON file allows for version control, review, and approval using a GitOps workflow. Additionally, when Apple releases new features (keys), administrators do not need to wait for a Fleet release to support these new features.

Learn more about customizing the [macOS Setup Assistant](https://fleetdm.com/docs/using-fleet/mdm-macos-setup-experience#macos-setup-assistant) experience.


## More new features, improvements, and bug fixes

* Added support to add an EULA as part of the AEP/DEP unboxing flow.
* DEP enrollments configured with SSO now pre-populate the username/fullname fields during account creation.
* Integrated the macOS setup assistant feature with Apple DEP so that the setup assistants are assigned to the enrolled devices.
* Re-assign and update the macOS setup assistants (and the default one) whenever required, such as when it is modified, when a host is transferred, a team is deleted, etc.
* Added device-authenticated endpoint to signal the Fleet server to send a webhook request with the device UUID and serial number to the webhook URL configured for MDM migration.
* Added UI for new automatic enrollment under the integration settings.
* Added UI for end-user migration setup.
* Changed macOS settings UI to always show the profile status aggregate data.
* Revised validation errors returned for `fleetctl mdm run-command`.
* Added `mdm.macos_migration` to app config.
* Added `PATCH /mdm/apple/setup` endpoint.
* Added `enable_end_user_authentication` to `mdm.macos_setup` in global app config and team config objects.
* Now tries to infer the bootstrap package name from the URL on upload if a content-disposition header is not provided.
* Added wildcards to host search so when searching for different accented characters you get more results.
* Can now reorder (and bookmark) policy tables by failing count.
* On the login and password reset pages, added email validation and fixed some minor styling bugs.
* Ensure sentence casing on labels in the host details page.
* Fix 3 Windows CIS benchmark policies that had false positive results initially merged on March 24.
* Fix of Fleet Server returning a duplicate OS version for Windows.
* Improved loading UI for disk encryption controls page.
* The 'GET /api/v1/fleet/hosts/{id}' and 'GET /api/v1/fleet/hosts/identifier/{identifier}' now include the software installed path on their payload.
* Third-party vulnerability integrations now include the installed path of the vulnerable software on each host.
* Greyed out unusable select all queries checkbox.
* Added page header for macOS updates UI.
* Back to queries button returns to previous table state.
* Bookmarkable URLs now source of truth for Manage Queries page table state.
* Added mechanism to refetch MDM enrollment status of a host pending unenrollment (due to a migration to Fleet) at a high interval.
* Made sure every modal in the UI conforms to a consistent system of widths.
* Team admins and team maintainers cannot save/update a global policy so hide the save button when viewing or running a global policy.
* Policy description has text area instead of one-line area.
* Users can now see the filepath of software on a host.
* Added version info metadata file to Windows installer.
* Fixed a bug where policy automations couldn't be updated without a webhook URL.
* Fixed tooltip misalignment on software page.

## Ready to upgrade?

Visit our [Upgrade guide](https://fleetdm.com/docs/deploying/upgrading-fleet) in the Fleet docs for instructions on updating to Fleet 4.32.0.

<meta name="category" value="releases">
<meta name="authorFullName" value="JD Strong">
<meta name="authorGitHubUsername" value="spokanemac">
<meta name="publishedOn" value="2023-05-24">
<meta name="articleTitle" value="Fleet 4.32.0 | User migration, customizing macOS Setup Assistant.">
<meta name="articleImageUrl" value="../website/assets/images/articles/fleet-4.32.0-1600x900@2x.png">
