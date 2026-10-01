# Apple Account restrictions on MDM-enrolled devices (research)

Tested August 2026. Updated September 2026 with **Allow Managed Apple Account on** results.

## Question

As asked by a customer:

> How can I block personal iCloud sign-ins on **any** MDM-enrolled device, and still allow Managed Apple Accounts, plus not allow Managed Apple Accounts outside of MDM-enrolled devices?

That is two restrictions pointing in opposite directions:

1. **On a managed device, only org accounts** — block personal iCloud, keep Managed Apple Accounts working.
2. **A Managed Apple Account, only on managed devices** — stop it signing in on a device the org doesn't manage.

Note the "any MDM-enrolled device" in the ask. It holds for restriction #2 but not for restriction #1: blocking personal iCloud keys off membership in Apple Business Manager (ABM) rather than MDM enrollment, while confining a Managed Apple Account to managed devices worked on every enrollment type tested, provided **Allow Managed Apple Account on** is set to `Managed devices only`. Both were investigated: what ABM can already do, and how the MDM `GetToken` check-in message with `TokenServiceType` of `com.apple.maid` fits in.

## What Apple Business Manager controls

Under **Settings → Access Management** there are two distinct settings that are easy to confuse:

| Setting | Direction | What it does |
|---|---|---|
| Apple Account on Organization Devices | Which account types may sign in on an org device | Restricting this to Managed Apple Accounts blocks personal iCloud sign-in |
| Allow Managed Apple Account on | Which devices a Managed Apple Account may sign in to | Governs restriction #2. Values: `Any device`, `Managed devices only`, `Supervised devices only`. Enforced through the `GetToken` check-in, see below |

Observed behavior when restricting **Apple Account on Organization Devices** to Managed Apple Accounts:

- It works. Sign-in with the org's Managed Apple Account succeeds; personal iCloud is refused.
- Propagation takes a few minutes — devices continue to accept personal sign-in immediately after the change.
- It only affects **new** sign-in attempts. Accounts already signed in are unaffected.
- It is a **global** setting. There is no per-group or per-device scoping.
- It requires Managed Apple Accounts, which requires the org to have captured its domains.

**Critically, "organization device" means a device in Apple Business Manager.** In testing, ADE-enrolled and supervised devices were restricted as expected. Devices enrolled via manual profile (OTA) enrollment could still sign in with a personal iCloud account, and Account-Driven User Enrollment (ADUE) was likewise unaffected.

## GetToken / com.apple.maid test results

The theory was that Fleet could gate Apple Account sign-in by declaring the `com.apple.mdm.token` server capability in the MDM enrollment payload and answering the resulting [`GetToken`](https://developer.apple.com/documentation/devicemanagement/get-token#Discussion) check-in for `com.apple.maid`.

It holds, but only for Managed Apple Accounts. The capability was added and the enrollment profile re-issued. ✅ means the device sent a `GetToken` check-in with `TokenServiceType` of `com.apple.maid`; ❌ means it never did, so that path cannot be used as a block.

| Enrollment | Managed Apple Account | Personal iCloud | Org email, not managed |
|---|---|---|---|
| Manual / OTA (iPhone) | ✅ | ❌ | ❌ |
| ADE (MacBook) | ✅ | ❌ | ❌ |
| ADUE | ✅ | ❌ | ❌ |

Device-channel request:

```xml
<dict>
	<key>MessageType</key>
	<string>GetToken</string>
	<key>TokenServiceType</key>
	<string>com.apple.maid</string>
	<key>UDID</key>
	<string>00006456-00023A461E7B801E</string>
</dict>
```

On macOS the same request arrives on the user channel, carrying `UserID`, `UserLongName`, `UserShortName`, and `NotOnConsole`.

ADUE sends it later in the flow than the other two: sign-in proceeds to the Remote Management screen, and `GetToken` fires once the user taps **Allow**.

That ordering has a consequence worth calling out. Enrollment has to complete before `GetToken` can fire, so an ADUE device really does become an enrolled host, and it stays one only if the Managed Apple Account sign-in succeeds. When the sign-in is refused, the device unenrolls itself about ten seconds later. An ADUE host can therefore appear in Fleet and then vanish on its own. That is the expected outcome of a refused sign-in, not a bug.

Whether the server's answer changes anything depends entirely on the **Allow Managed Apple Account on** setting: under one of its three values it decides the sign-in, and under the other two it is ignored. See the next section.

## Results by "Allow Managed Apple Account on"

Apple enforces restriction #2 through `GetToken`. The device asks its MDM server for a `com.apple.maid` token and Apple verifies that token before letting the Managed Apple Account sign in. Fleet answers with an RS256 JWT signed by the ABM private key, carrying `service_type: com.apple.maid` and an `iss` claim of the ABM server UUID for the token that enrolled the host (`server/service/apple_mdm_checkin_protocol.go`).

All three values were tested. Only one of them acts on what Fleet returns:

| Setting value | What decides the sign-in | Does Fleet's answer matter? |
|---|---|---|
| `Managed devices only` | Whether Fleet returns a token Apple can verify | Yes. It is the deciding factor |
| `Supervised devices only` | Whether the device is supervised | No. Apple requests the token, then refuses anything unsupervised regardless |
| `Any device` | Nothing. Every sign-in succeeds | No. The response is fetched and ignored |

So `Managed devices only` is the only value under which the `GetToken` handler changes an outcome. Under the other two Apple still sends the check-in, which makes the request's presence a poor signal that anything is being enforced.

### Managed devices only

Only Managed Apple Account sign-ins were tested. Personal iCloud and unmanaged org email are out of scope for this setting: `GetToken` never fires for them (see the table above).

| Device | Correctly signed token | Wrongly signed token | Empty, non-error response |
|---|---|---|---|
| Supervised Mac, registered in ABM | Signs in | Refused, "Verification failed" | Refused, same "Verification failed" |
| Non-supervised iPhone, manual enrollment | Signs in | Refused, same "Verification failed" | Refused, same "Verification failed" |
| iPhone, ADUE | Signs in | Refused, after the user taps **Allow** on Remote Management | Refused, same error as a wrongly signed token |

"Wrongly signed" means the JWT carried a server UUID other than the one Apple holds for the MDM server the device is enrolled in.

Supervised Mac, wrongly signed token:

![Verification failed on a supervised Mac][signed-with-wrong-token-mac]

ADUE, wrongly signed token:

![Sign-in failed during ADUE][signed-with-wrong-token-adue]

A device with no MDM enrollment never reaches Fleet at all. No `GetToken` check-in arrives, which follows from the mechanism: with no MDM server enrolled, the device has nowhere to ask. Apple refuses the sign-in on its own.

![Verification failed on a non-managed device][non-managed-device-maa-failure]

Every refusal reports the platform's "Verification Failed" dialog, with one exception:

| Case | Dialog |
|---|---|
| Supervised Mac, wrongly signed token or empty response | "Verification Failed", "You can't sign into this device using this Apple ID. Contact your organization's administrator for assistance." |
| Non-supervised iPhone with manual enrollment, wrongly signed token or empty response | The same dialog in iOS wording: "You cannot sign into this device using this Apple ID. Contact your organisation's administrator for assistance." |
| Non-managed iPhone | Identical to the row above |
| iPhone, ADUE, wrongly signed token or empty response | "Sign-in Failed", "The operation couldn't be completed. (AKAuthenticationError error -7013.)", over the Configuring iPhone screen |

Two consequences for support:

- ADUE is the exception. It surfaces an opaque `AKAuthenticationError -7013` rather than admin-facing wording, so a policy refusal there reads as a generic sign-in problem.
- The reason for a refusal is never visible on the device. A wrong server UUID, an empty response, and no MDM enrollment at all produce the same dialog on a given platform, so diagnosis has to start from the Fleet server logs, where a missing `GetToken` check-in is itself the signal that the device is not enrolled.

Four things follow:

- Enforcement does not depend on supervision or on ABM device membership. A non-supervised, manually enrolled iPhone was gated exactly like a supervised ABM Mac.
- ADUE is gated as well, even though its check-in arrives only after the user accepts management.
- **This setting fails closed on its own.** A correct response is required, not merely preferred. On all three enrollment types an empty, non-error response was refused exactly like an unverifiable token, so anything short of a token Apple can verify blocks the account.
- **Restriction #2 is satisfied by this setting.** A Managed Apple Account cannot sign in on an unmanaged device.

#### What this means for Fleet

- **Restriction #2 runs through Fleet.** The account signs in only if Fleet returns a token Apple can verify. The enrollment profile templates declare `com.apple.mdm.token` alongside `com.apple.mdm.per-user-connections` and `com.apple.mdm.bootstraptoken` (`server/mdm/apple/apple_mdm.go`), and `MDMAppleGetTokenService` answers the check-in (`server/service/apple_mdm_checkin_protocol.go`).
- **The unmanaged-device half needs no Fleet work.** Apple refuses the account on a device with no MDM enrollment before any server is contacted, so Fleet only matters for devices that are enrolled.
- **Fleet does not have to build a denial path.** Because the setting fails closed, not issuing a verifiable token is already a refusal. The allow path is the one that has to work: return a JWT signed with the right ABM server UUID for that host.
- **The failure mode to watch is the inverse.** Any host Fleet cannot resolve a server UUID for, meaning no ABM token, no DEP assignment and no default token, or an ABM token whose `server_uuid` was never fetched, gets an error response from `MDMAppleGetTokenService`. This is untested against this value. An error response was tried only under `Any device`, where it permitted the sign-in, and that says nothing about what happens here. Given an empty response is refused, an error most likely is too, which would mean a Fleet misconfiguration blocks Managed Apple Account sign-in outright. Worth confirming before shipping.

### Supervised devices only

| Device | Correctly signed token | Wrongly signed token | Empty, non-error response |
|---|---|---|---|
| Supervised Mac, registered in ABM | Signs in | Refused, "Verification failed" | Refused, same "Verification failed" |
| Non-supervised iPhone, manual enrollment | Refused, "Verification failed" | Refused, same dialog | Refused, same dialog |
| iPhone, ADUE | Refused, `AKAuthenticationError -7013` | Refused, same error | Refused, same error |

Supervision is the only thing that decides an outcome here. The supervised Mac behaves exactly as it does under `Managed devices only`. Every device that is not supervised is refused, and a correctly signed token does not help.

A device with no MDM enrollment is refused here too, with the same "Verification Failed" dialog it gets under `Managed devices only`. No `GetToken` fires, for the same reason as before: the device has no MDM server to ask.

`GetToken` still arrives on the refused devices that are enrolled. The non-supervised iPhone asked Fleet for a token and Fleet answered normally; Apple refused the sign-in afterward. So Apple asks even when it has already decided to say no, and a refusal under this setting leaves a successful issued-token line in the Fleet logs.

The dialogs match the ones documented above: the platform's "Verification Failed" dialog everywhere except ADUE, which again reports `AKAuthenticationError -7013`.

Two things follow:

- **This setting enforces supervision, not enrollment.** The non-supervised, manually enrolled iPhone signed in under `Managed devices only` with a correctly signed token. Here the same device with the same correct token is refused.
- **ADUE and Managed Apple Accounts are mutually exclusive under this setting.** Account-driven user enrollment never produces a supervised device, so no ADUE device can sign in to a Managed Apple Account no matter what Fleet answers.

#### What this means for Fleet

- **Fleet's answer stops mattering for anything not supervised.** A correctly signed token is refused exactly like a wrong one, so there is no server-side fix for a non-supervised device under this setting. Supervision is the only lever, which makes this the one setting value where the `GetToken` handler cannot affect the outcome.
- **The logs read as a success.** Since Apple asks for the token before refusing, Fleet records issuing a valid one for a sign-in that failed. Under `Managed devices only` an absent check-in was the useful signal; here the check-in is present and says nothing about the outcome, so the Fleet logs cannot be used to diagnose these refusals.
- **Expect ADUE hosts that enroll and immediately unenroll.** Every ADUE sign-in attempt is refused under this setting, and each one leaves the enroll-then-unenroll trace described above.

### Any device

This value imposes no restriction, and Fleet's answer is ignored entirely.

| Device | Result |
|---|---|
| iPhone, ADUE, correctly signed token | Signs in |
| iPhone, ADUE, wrongly signed token | Signs in. The device fetches the token and does not act on it |
| iPhone, ADUE, empty non-error response | Signs in |
| Any enrolled device, error response | Signs in |
| Device with no MDM enrollment | Signs in. No `GetToken` fires |

The supervised Mac and the manually enrolled iPhone were not run separately. ADUE and the non-enrolled device are the most constrained cases in the matrix, and both signed in regardless of what Fleet returned, so the less constrained ones follow.

Two things follow:

- **This value explains an earlier contradiction in this document.** The first round of testing concluded that an absent or empty `GetToken` response still permits sign-in. That is true, but only here. The finding was setting-dependent rather than wrong, and the original testing was evidently done with this value in effect.
- **An error response is not a denial either.** This is the only value where the error paths in `MDMAppleGetTokenService` were exercised, and they did not block the sign-in.

#### What this means for Fleet

- **Nothing Fleet does has any effect.** A wrong server UUID, an empty response, an error, and no MDM enrollment at all produce the same outcome. The `GetToken` handler cannot be tested meaningfully against this value.
- **ADUE leaves no enroll-then-unenroll trace here.** The sign-in succeeds, so the device stays enrolled. That trace appears only when a sign-in is refused.

## Gaps

- `GetToken` fires **only** for Managed Apple Accounts. Personal iCloud and unmanaged org email never trigger it, so it cannot be used to block personal accounts — under any enrollment type.
- That leaves no selective MDM-side lever for blocking personal iCloud on manual/OTA and ADUE enrollments: they are outside the reach of **Apple Account on Organization Devices**, and only the all-or-nothing restriction below reaches them.
- **Apple Account on Organization Devices** cannot be scoped. Turning it on is all-or-nothing for the organization.

## `allowAccountModification` (Restrictions payload)

The [Restrictions payload](https://developer.apple.com/documentation/devicemanagement/restrictions) has an `allowAccountModification` key. Setting it to `false` blocks **all** iCloud sign-ins on the device. Unlike the ABM setting, it cannot be bypassed for Managed Apple Accounts — the org's own accounts are blocked too.

It also blocks adding any other internet account: mail, contacts, calendars, and so on.

It reaches devices the ABM setting does not, but it answers "no accounts at all," not "only our accounts."

## Implications for Fleet

Findings that hold only for a particular **Allow Managed Apple Account on** value are recorded with that value above.

- **Restriction #1 needs no Fleet server work.** It is an ABM setting, subject to the limits above: ABM/ADE + supervised only, global scope, new sign-ins only, Managed Apple Accounts required.
- **The `GetToken` handler earns its keep under exactly one setting value.** `Managed devices only` is the only value that acts on Fleet's response. `Supervised devices only` requests the token and then refuses anything unsupervised anyway, and `Any device` ignores the response. Anyone testing the handler has to set the value to `Managed devices only` first, or every result will look like a pass.
- Signing requires an ABM token that has Apple's `server_uuid`. Fleet takes the ABM token from the host's DEP assignment and falls back to the default ABM token for hosts that are not DEP-assigned; with no server UUID, no token can be issued.
- Both restrictions depend on Managed Apple Accounts, which requires captured domains.

## References

- [`GetToken` check-in message](https://developer.apple.com/documentation/devicemanagement/get-token) — Apple Developer documentation
- [Customize user access to apps and services](https://support.apple.com/en-gb/guide/business/axm53xk34bq/1/web/1) — the Apple Business Manager settings described above
- [Restrictions payload](https://developer.apple.com/documentation/devicemanagement/restrictions) — `allowAccountModification` and the other restriction keys

[signed-with-wrong-token-mac]: ../../assets/wrong-signing-token-macos.png
[signed-with-wrong-token-adue]: ../../assets/wrong-signing-token-adue.jpg
[non-managed-device-maa-failure]: ../../assets/non-managed-device-maa-failure.jpg
