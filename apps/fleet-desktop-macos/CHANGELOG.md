## Fleet Desktop macOS v1.6.0 (November 13, 2026)

* Fleet Desktop now runs on Macs that aren't MDM-enrolled. When the fleetd configuration profile isn't present, the app reads the Fleet URL from the orbit launchd plist (`ORBIT_FLEET_URL`), and the installer no longer requires the managed preferences profile.
* The `notify` command now exits with code 50 when another notification is already displayed, so Fleet can retry later instead of stacking toasts.
* Pressing Cmd+Shift+X now closes a `notify` toast, as a fallback when the page's own dismiss doesn't work.
* Fixed end users getting stuck on the MDM enrollment page with no way back to **My device**. **Turn on MDM** now opens the enrollment page in a sheet over **My device** with a **Close** button, shows every enrollment step without scrolling, and stays open after the profile downloads.
* Fixed Okta FastPass sign-in when Fleet Desktop SSO is enabled. Okta launches Okta Verify through its `com-okta-authenticator` URL scheme, which the in-app browser now hands to the app instead of abandoning the sign-in and showing "Couldn't sign in."
* Fixed "Read about transparency" and "About Fleet" links opening inside Fleet Desktop with no way to navigate back. They now open in the default web browser.
* Fixed the installer package containing a second copy of Fleet Desktop.app, which roughly halves its size.

## Fleet Desktop macOS v1.5.1 (September 28, 2026)

* Added a `notify` command to Fleet Desktop for showing patch notification toasts.
* The Dock badge now counts only failing policies that are shown to the end user, ignoring policies hidden by the IT admin.
* Fixed Platform SSO user registration on Macs that were set up before the PSSO profile was installed. The extension now prompts for IdP credentials and verifies them through Fleet instead of failing registration immediately.

## Fleet Desktop macOS v1.4.0 (July 24, 2026)

* Migrated Fleet Desktop macOS from allenhouchins/fleet-desktop to fleetdm/fleet repo
* Added a Platform SSO Extension that when used with Fleet's Apple Account Provisioning feature allows password synchronization and account provisioning during Setup Assistant with any Oauth ROPG supporting IdP
