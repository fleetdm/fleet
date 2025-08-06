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

## POC details

* A new `GET /api/v1/fleet/mdm/sso` endpoint was needed to automatically initiate the SSO session with a simple HTTP redirect (only a `POST` endpoint already existed, which requires client-side javascript to trigger).
* Most of the existing MDM SSO logic works as-is for this use-case, but it is named as "MDMAppleSSO" (e.g. `InitiateMDMAppleSSO`, `MDMAppleSSOCallback`). Some refactoring will be needed to make this more platform-agnostic (we probably want to maintain the existing endpoints and add new ones, as the SSO callback URL - the `AssertionConsumerService` - is part of the SAML config and we don't want to break existing configs).
* [Error-handling in the SSO flow](https://github.com/fleetdm/fleet/pull/31651/files#diff-4fe044b62304109be6c303bb0dec9d0151c6ce84f83e3352bfb316da3889920dR765-R769) needs to be adapted for this use-case, as currently it always redirects to `apple_mdm.FleetUISSOCallbackPath + "?error=true"` (which does not make sense in our case, we probably want to redirect to `/enroll` again with some query string to indicate the error).
* The [existing SSO endpoints](https://github.com/fleetdm/fleet/pull/31651/files#diff-9aab42757aa328e6c16e607951a4b81086f9caae7c4a087d4494a821c7b9470cR1038-R1043) are in the `neAppleMDM` router, which checks for Apple MDM being enabled in a middleware. We probably want the endpoints for our case to be in a router that doesn't check for any MDM being enabled (as we have specific error pages for each case in the figma).
* Using the `RelayState` query parameter to pass the original `/enroll` path with the enroll secret worked well. We also have the `originalURL` saved as state in the SSO JSON session, so we could do without the relay state I think. For the POC I stored `/enroll` (no query string) as the original URL, and passed the full path+query string in the relay state.
* The frontend needs [additional changes](https://github.com/fleetdm/fleet/pull/31651/files#diff-72f7403682d211fc8a84a411fc39c4a33c3eb6a33549a33f1179dd7da6a893ccR962-R963) to show the `Users` card and [the IdP `Username` (but without the groups/departments/other sub-sections)](https://github.com/fleetdm/fleet/pull/31651/files#diff-f70cfae61296b0db85ed625ad106c9481407c3b33345b6b566a57c10a8dab45aR53), as currently it hides those on iDevices.
* Found that [the `HostLite` struct](https://github.com/fleetdm/fleet/pull/31651/files#diff-b6d4c48f2624b82c2567b2b88db1de51c6b152eeb261d40acfd5b63a890839b7R1416) needed a small change to account for loading hosts with a `NULL` osquery id.
* On successful IdP login, a [cookie is created to store the IdP account's uuid](https://github.com/fleetdm/fleet/pull/31651/files#diff-1cc547279489e5119326f2ac15610c2dd7519ef7dada5952552765cefabd41aaR3345-R3348), which serves as indicator that the user did authenticate successfully and tracks the matching account even if the page is refreshed (for 30 minutes). I don't think there's any security concern with this (cookie is HTTP-only, requires https and uses the `__Host-` prefix), and the IdP UUID is not sensitive information.
* A resulting edge-case of this is that if the user authenticated with the wrong account, they have to delete the cookies and refresh the page to login again (or wait 30 minutes). I think this is fine, otherwise we could have a "logout" link or button on the enroll page.

## New or updated sub-tasks

It is a bit tricky to update the sub-tasks of this story as they have already been defined and estimated, so any important change could change the estimation. We can worry about this if we feel like the estimation is no longer valid, but regarding the sub-tasks I'll go with some recommendations here:


## Tasks not covered by the POC

Some tasks do not represent a big risk / have no big unknowns and as such, have not been covered by the POC. These include:

* The check for "any team with IdP enabled" and whether the enroll secret matches a known team or not in the initial /enroll call (simple DB lookups).
* The various error states in the UI (e.g. iDevice when Apple MDM is off, etc.).
