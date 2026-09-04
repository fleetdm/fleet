# Blocking personal Apple Accounts on MDM-enrolled devices (research)

Tested August 2026.

## Question

As asked by a customer:

> How can I block personal iCloud sign-ins on **any** MDM-enrolled device, and still allow Managed Apple Accounts, plus not allow Managed Apple Accounts outside of MDM-enrolled devices?

That is two restrictions pointing in opposite directions:

1. **On a managed device, only org accounts** — block personal iCloud, keep Managed Apple Accounts working.
2. **A Managed Apple Account, only on managed devices** — stop it signing in on a device the org doesn't manage.

Note the "any MDM-enrolled device" in the ask. It is the part that does not hold: as the findings below show, the available controls key off membership in Apple Business Manager (ABM), not off MDM enrollment. Both were investigated — what ABM can already do, and whether the MDM `GetToken` check-in message with `TokenServiceType` of `com.apple.maid` closes the gaps.

## What Apple Business Manager controls

Under **Settings → Access Management** there are two distinct settings that are easy to confuse:

| Setting | Direction | What it does |
|---|---|---|
| Apple Account on Organization Devices | Which account types may sign in on an org device | Restricting this to Managed Apple Accounts blocks personal iCloud sign-in |
| Allow Managed Apple Account on | Which devices a Managed Apple Account may sign in to | Governs restriction #2 |

Observed behavior when restricting **Apple Account on Organization Devices** to Managed Apple Accounts:

- It works. Sign-in with the org's Managed Apple Account succeeds; personal iCloud is refused.
- Propagation takes a few minutes — devices continue to accept personal sign-in immediately after the change.
- It only affects **new** sign-in attempts. Accounts already signed in are unaffected.
- It is a **global** setting. There is no per-group or per-device scoping.
- It requires Managed Apple Accounts, which requires the org to have captured its domains.

**Critically, "organization device" means a device in Apple Business Manager.** In testing, ADE-enrolled and supervised devices were restricted as expected. Devices enrolled via manual profile (OTA) enrollment could still sign in with a personal iCloud account, and Account-Driven User Enrollment (ADUE) was likewise unaffected.

## GetToken / com.apple.maid test results

The theory was that Fleet could intercept Apple Account sign-in by declaring the `com.apple.mdm.token` server capability in the MDM enrollment payload and refusing the resulting [`GetToken`](https://developer.apple.com/documentation/devicemanagement/get-token#Discussion) check-in for `com.apple.maid`.

The capability was added and the enrollment profile re-issued. ✅ means the device sent a `GetToken` check-in with `TokenServiceType` of `com.apple.maid`; ❌ means it never did, so that path cannot be used as a block.

| Enrollment | Managed Apple Account | Personal iCloud | Org email, not managed |
|---|---|---|---|
| Manual / OTA (iPhone) | ✅ | ❌ | ❌ |
| ADE (MacBook) | ✅ | ❌ | ❌ |
| ADUE | ❌ | ❌ | ❌ |

Device-channel request:

```xml
<dict>
	<key>MessageType</key>
	<string>GetToken</string>
	<key>TokenServiceType</key>
	<string>com.apple.maid</string>
	<key>UDID</key>
	<string>00008110-00025D461E7B801E</string>
</dict>
```

On macOS the same request arrives on the user channel, carrying `UserID`, `UserLongName`, `UserShortName`, and `NotOnConsole`.

Two further findings:

- Returning nothing for a `GetToken` request still allows the sign-in to proceed. An empty or absent response is not a denial.
- ADUE devices never send `GetToken` at all, for any account type.

## Gaps

- `GetToken` fires **only** for Managed Apple Accounts. Personal iCloud and unmanaged org email never trigger it, so it cannot be used to block personal accounts — under any enrollment type.
- Manual/OTA enrollments and ADUE are outside the reach of the ABM setting, and `GetToken` offers no substitute. **No selective MDM-side lever blocks personal iCloud on those devices** — only the all-or-nothing restriction below.
- The ABM setting cannot be scoped. Turning it on is all-or-nothing for the organization.

## `allowAccountModification` (Restrictions payload)

The [Restrictions payload](https://developer.apple.com/documentation/devicemanagement/restrictions) has an `allowAccountModification` key. Setting it to `false` blocks **all** iCloud sign-ins on the device. Unlike the ABM setting, it cannot be bypassed for Managed Apple Accounts — the org's own accounts are blocked too.

It also blocks adding any other internet account: mail, contacts, calendars, and so on.

It reaches devices the ABM setting does not, but it answers "no accounts at all," not "only our accounts."

## Implications for Fleet

- **Restriction #1 needs no Fleet server work.** It is an ABM setting, subject to the limits above: ABM/ADE + supervised only, global scope, new sign-ins only, Managed Apple Accounts required.
- **Restriction #2 is the piece that would need implementation.** Fleet would have to declare `com.apple.mdm.token` in `ServerCapabilities` and answer `GetToken` for `com.apple.maid`. Today the enrollment payload declares only `com.apple.mdm.per-user-connections` and `com.apple.mdm.bootstraptoken` (`server/mdm/apple/apple_mdm.go`). The nanomdm layer already parses the `GetToken` message and its response type (`server/mdm/nanomdm/mdm/checkin.go`); the handling and policy are what's missing.
- Any denial response must be explicit. Since an empty response permits sign-in, "fail closed" has to be deliberate.
- Both restrictions depend on Managed Apple Accounts, which requires captured domains.

## References

- [`GetToken` check-in message](https://developer.apple.com/documentation/devicemanagement/get-token) — Apple Developer documentation
- [Customize user access to apps and services](https://support.apple.com/en-gb/guide/business/axm53xk34bq/1/web/1) — the Apple Business Manager settings described above
- [Restrictions payload](https://developer.apple.com/documentation/devicemanagement/restrictions) — `allowAccountModification` and the other restriction keys
