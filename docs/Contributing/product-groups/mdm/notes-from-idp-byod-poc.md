# Findings and notes from the IDP BYOD POC

This doc (and the whole branch/PR) is not meant to be merged, I'm using a markdown file to make reviewing/commenting easier.

## Local development

Our docker setup includes a SimpleSAML implementation that can be used for local development and testing. However, to enroll a mobile device, we need public URLs (e.g. via `ngrok`), and valid corresponding SimpleSAML configuration.

To this end, I had to add this section to the SimpleSAML config: https://github.com/fleetdm/fleet/pull/31651/files#diff-6e61c88395df7b3b9ac322f43f820b8eced754487191dfc752a1bc24ead70956R11

Which maps to my ngrok-exposed URL for SimpleSAML:

```
# ngrok config
endpoints:
  # ...
  - name: idp
    url: https://mnaidp.ngrok.app
    upstream:
      url: http://localhost:9080
```

And then run `ngrok start fleet idp`. In Fleet Settings -> Integrations -> MDM (which will eventually be moved to IdP sub-section as part of this story), the IdP is configured as:

* Provider name: SimpleSAML
* Entity ID: https://mnafleet.ngrok.app (must map to the SimpleSAML config file)
* Metadata URL: https://mnaidp.ngrok.app/simplesaml/saml2/idp/metadata.php

And then in Controls -> Setup experience -> End user authentication, turn it on for a team.

Note that for the POC, the `/enroll` logic simply looked at the enroll secret and triggered the IdP-based flow if it starts with `idpteam`.

I tested with an iPhone device and an Android device.

## POC findings/takeaways/notes

* A new `GET /api/v1/fleet/mdm/sso` endpoint was needed to automatically initiate the SSO session with a simple HTTP redirect (only a `POST` endpoint already existed, which requires client-side javascript to trigger).
* Most of the existing MDM SSO logic works as-is for this use-case, but it is named as "MDMAppleSSO" (e.g. `InitiateMDMAppleSSO`, `MDMAppleSSOCallback`). Some refactoring will be needed to make this more platform-agnostic (we probably want to maintain the existing endpoints and add new ones, as the SSO callback URL - the `AssertionConsumerService` - is part of the SAML config and we don't want to break existing configs).
* [Error-handling in the SSO flow](https://github.com/fleetdm/fleet/pull/31651/files#diff-4fe044b62304109be6c303bb0dec9d0151c6ce84f83e3352bfb316da3889920dR765-R769) needs to be adapted for this use-case, as currently it always redirects to `apple_mdm.FleetUISSOCallbackPath + "?error=true"` (which does not make sense in our case, we probably want to redirect to `/enroll` again with some query string to indicate the error).
* The [existing SSO endpoints](https://github.com/fleetdm/fleet/pull/31651/files#diff-9aab42757aa328e6c16e607951a4b81086f9caae7c4a087d4494a821c7b9470cR1038-R1043) are in the `neAppleMDM` router, which checks for Apple MDM being enabled in a middleware. We probably want the endpoints for our case to be in a router that doesn't check for any MDM being enabled (as we have specific error pages for each case in the figma).
* Using the `RelayState` query parameter to pass the original `/enroll` path with the enroll secret worked well. We also have the `originalURL` saved as state in the SSO JSON session, so we could do without the relay state I think. For the POC I stored `/enroll` (no query string) as the original URL, and passed the full path+query string in the relay state.
* The frontend needs [additional changes](https://github.com/fleetdm/fleet/pull/31651/files#diff-72f7403682d211fc8a84a411fc39c4a33c3eb6a33549a33f1179dd7da6a893ccR962-R963) to show the `Users` card and [the IdP `Username` (but without the groups/departments/other sub-sections)](https://github.com/fleetdm/fleet/pull/31651/files#diff-f70cfae61296b0db85ed625ad106c9481407c3b33345b6b566a57c10a8dab45aR53), as currently it hides those on iDevices.
* Found that [the `HostLite` struct](https://github.com/fleetdm/fleet/pull/31651/files#diff-b6d4c48f2624b82c2567b2b88db1de51c6b152eeb261d40acfd5b63a890839b7R1416) needed a small change to account for loading hosts with a `NULL` osquery id.
* On successful IdP login, a [cookie is created to store the IdP account's uuid](https://github.com/fleetdm/fleet/pull/31651/files#diff-1cc547279489e5119326f2ac15610c2dd7519ef7dada5952552765cefabd41aaR3345-R3348), which serves as indicator that the user did authenticate successfully and tracks the matching account even if the page is refreshed (for 30 minutes). I don't think there's any security concern with this (cookie is HTTP-only, requires https and uses the `__Host-` prefix), and the IdP UUID is not sensitive information.
* A resulting edge-case of this is that if the user authenticated with the wrong account, they have to delete the cookies and refresh the page to login again (or wait 30 minutes). I think this is fine, otherwise we could have a "logout" link or button on the enroll page.
* As [explicitly requested by Marko](https://fleetdm.slack.com/archives/C03C41L5YEL/p1754390071596379?thread_ts=1754329679.371229&cid=C03C41L5YEL), we want to maintain the same behaviour as before when BYOD-enrolling so that the enroll secret is validated only at profile installation time, so if an unknown/invalid enroll secret is passed to `/enroll` initially, we go through the SSO login if SSO is enabled for at least one team, and proceed with downloading the profile if the login is correct, and only fail at profile install time on the device regarding the enroll secret validation.

For iOS/iPadOS, the flow is as follows:

1. `GET /enroll?enroll_secret=...` gets requested (implemented in `server/service/frontend.go`, `ServeEndUserEnrollOTA`). 
	* In the actual implementation, this is where the team lookup for the enroll secret would happen: if the team has IdP enabled, redirect to SSO, if not, proceed without SSO, and if the enroll secret is invalid, redirect to SSO if any team has IdP enabled, otherwise proceed without SSO.
	* For the POC, if the enroll secret starts with `idpteam`, it starts the SSO flow with an HTTP redirect.
	* Redirect to `/api/latest/fleet/mdm/sso` with `initiator=ota_enroll` and `RelayState=/enroll?enroll_secret=...` as query strings.
2. `GET /api/latest/fleet/mdm/sso` gets called with the `initiator` and `RelayState`. Based on the `initiator` value, set the `originalURL` argument to the `RelayState` that was received (the POC used a slightly different approach with `originalURL=/enroll` and `RelayState=/enroll?enroll_secret=...`, but technically we shouldn't need both at this point, only `originalURL` would be enough). It then redirects to the configured SSO provider.
3. `POST /api/latest/fleet/mdm/sso/callback` receives the SAML response (and `RelayState` if we were to use it). It handles validation of the response and based on the `originalURL` (saved in the Redis session associated with the SSO session, which gets stored in a cookie), detects that it is a BYOD enrollment and redirects to `/enroll` with the original enroll secret, the enrollment reference (which matches the IdP account UUID), and stores that IdP account UUID in a cookie valid 30 minutes, to store client-side that the user is already logged in case of a page refresh.
4. `GET /enroll?enroll_secret=...&enrollment_reference=...` gets called once again. This time, it detects that the "SSO authenticated" cookie is present so it does not redirect to the SSO flow, instead it renders the page with the Download profile button.
5. `GET /enrollment_profiles/ota` gets called with the enrollment secret and (if SSO was done) the cookie that contains the IdP account UUID. It generates the enrollment profile with both of those identifiers in the enrollment URL as query string parameters (`enroll_secret` and `idp_uuid`) and returns it so the device can download and install it.
6. `POST /ota_enrollment?enroll_secret=...&idp_uuid=...` gets called when the user installs the profile on the device. If `idp_uuid` is present, the [enrolled host gets associated with this IdP account](https://github.com/fleetdm/fleet/pull/31651/files#diff-1cc547279489e5119326f2ac15610c2dd7519ef7dada5952552765cefabd41aaR7113-R7123).

For Android, the flow is the same for steps 1-4. Since we don't associate the newly enrolled device with the IdP account on Android for now, the rest is as before, the SSO flow only serves as authentication for the user to download the enrollment profile.

## New or updated sub-tasks

It is a bit tricky to update the sub-tasks of this story as they have already been defined and estimated, so any important change could change the estimation. We can worry about this if we feel like the estimation is no longer valid, but regarding the sub-tasks I'll go with some recommendations here:

* There are two main parts on the backend: the pre-enroll-OTA logic (steps 1-4 above) and the OTA profile generation/enrollment/IdP association steps (5-6). 
	* I'd suggest merging [#30659](https://github.com/fleetdm/fleet/issues/30659) with [#30660](https://github.com/fleetdm/fleet/issues/30660) and make that the pre-enroll-OTA sub-task (biggest sub-task of the story).
	* I'd keep [#30661](https://github.com/fleetdm/fleet/issues/30661) as the OTA generation/enrollment sub-task. We only need to agree on the cookie name for both tasks to be addressed in parallel.
	* I'd delete [#30663](https://github.com/fleetdm/fleet/issues/30663), I don't see a need for it anymore (covered in the other two backend tasks).
* For the frontend, update [#30662](https://github.com/fleetdm/fleet/issues/30662) to include changes required to show the IdP Users card and Username information for iDevices.
* Add a new backend/frontend sub-task to cover the validations and frontend pages required for the various error states described in the Figma.
* Use the existing Guide updates ticket [#30684](https://github.com/fleetdm/fleet/issues/30684) to document the various enrollment flows as [suggested by Jordan](https://github.com/fleetdm/fleet/issues/30692#issuecomment-3140594238).

## Tasks not covered by the POC

Some tasks do not represent a big risk / have no big unknowns and as such, have not been covered by the POC. These include:

* The check for "any team with IdP enabled" and whether the enroll secret matches a known team or not in the initial /enroll call (simple DB lookups). This would be part of the pre-enroll-OTA sub-task.
* The various error states in the UI (e.g. iDevice when Apple MDM is off, etc.).
* Showing EULA for BYOD enrollment - this was not designed/spec'd as part of the story, but the [updated copy on the Setup experience page](https://www.figma.com/design/fw7XXg2QzBOa7YJ9r2Cchp/-29222-IdP-authentication-before-BYOD-iOS--iPadOS--and-Android-enrollment?node-id=5319-3602&t=aCqdNEzXdyrwuS0G-0) makes it sound like it would show the EULA. I [asked Marko about this](https://fleetdm.slack.com/archives/C03C41L5YEL/p1754492400598889) on slack, he confirmed that we don't show EULA for BYOD, there will be a copy change to avoid any confusion.
