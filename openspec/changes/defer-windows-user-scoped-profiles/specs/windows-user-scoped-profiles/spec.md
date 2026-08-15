## ADDED Requirements

### Requirement: Windows profile channel scope classification

Fleet SHALL classify every Windows configuration profile as device-scoped or user-scoped, derived from the canonicalized `Add` and `Replace` target LocURIs of the SyncML that Fleet actually delivers. A profile whose delivered SyncML contains at least one target resolving under `./User/` SHALL be classified user-scoped. Classification MUST use the same normalization and parser as the delivery path, so that classification and delivery cannot disagree.

#### Scenario: Profile with only device targets

- **WHEN** a profile's targets all resolve under `./Device/` or the scope-less `./Vendor/MSFT/` form
- **THEN** Fleet classifies the profile as device-scoped

#### Scenario: Profile with a user target

- **WHEN** a profile contains at least one target resolving under `./User/`
- **THEN** Fleet classifies the profile as user-scoped

#### Scenario: Scope-less and obfuscated spellings cannot bypass classification

- **WHEN** a profile expresses a `./User/` target using a spelling that the raw string form does not match literally, such as text split by CDATA sections, comments, or nested elements
- **THEN** Fleet still classifies the profile as user-scoped, because classification operates on the same canonicalized LocURIs the delivery path produces

#### Scenario: SCEP profiles are classified after atomic normalization

- **WHEN** a SCEP profile is classified
- **THEN** classification operates on the same atomic-wrapped bytes that delivery sends, so the nested commands inside the generated `Atomic` are inspected

### Requirement: Observing the device's MDM user context

Fleet SHALL read the OMA-DM generic alert `1224` carrying alert type `com.microsoft/MDM/LoginStatus` from incoming management sessions, and SHALL persist the reported value and its observation time on the enrollment. Fleet MUST treat an enrollment with no recorded observation as "never observed" rather than as an absence of user context.

#### Scenario: Session reports an MDM user is signed in

- **WHEN** a management session carries alert `1224` with type `com.microsoft/MDM/LoginStatus` and data `user`
- **THEN** Fleet records the enrollment's user context as present, with the observation time

#### Scenario: Session reports no active user

- **WHEN** a management session carries alert `1224` with type `com.microsoft/MDM/LoginStatus` and data `none`
- **THEN** Fleet records the enrollment's user context as absent, with the observation time

#### Scenario: Session reports a signed-in user without an MDM account

- **WHEN** a management session carries alert `1224` with type `com.microsoft/MDM/LoginStatus` and data `others`
- **THEN** Fleet records that value, and MUST NOT treat it as user context present

#### Scenario: Alert absent from the session

- **WHEN** a management session carries no `com.microsoft/MDM/LoginStatus` alert
- **THEN** Fleet leaves any previously recorded observation unchanged and does not record a new one

#### Scenario: Alert arrives in the message that Fleet does not otherwise persist

- **WHEN** the alert arrives in a message whose only status references the SyncML header, which Fleet does not store as a response
- **THEN** Fleet still records the observed user context, because the record is written independently of response persistence

#### Scenario: Alert is repeated in the authenticated message

- **WHEN** Fleet answers a session's first message with an authentication challenge, and the device repeats its alerts in the following authenticated message
- **THEN** Fleet records the observed user context from the authenticated message, and MUST NOT require alert processing on the unauthenticated challenge path

### Requirement: Deferring user-scoped profiles when a user context can still arrive

For an enrollment that has a bound MDM user identity and has not reported `user`, Fleet SHALL hold user-scoped profiles rather than delivering them. A held profile SHALL report status Pending with a detail explaining that Fleet is waiting for a user to sign in, and Fleet MUST NOT enqueue a command for it. Only a reported value of `user` SHALL release the hold: `others`, `none`, and the absence of any observation all keep the profile held. When a session subsequently reports `user`, Fleet SHALL deliver the held profile without operator action.

#### Scenario: User-scoped profile assigned during OOBE

- **WHEN** a user-scoped profile applies to a host whose enrollment has a UPN-backed user identity and whose latest session reported `others`, which is the value Windows reports during OOBE while the setup account is signed in
- **THEN** Fleet does not enqueue the profile, and the profile reports Pending with a detail explaining that it is waiting for first sign-in

#### Scenario: No user is signed in at all

- **WHEN** the enrollment has a UPN-backed user identity and its latest session reported `none`
- **THEN** Fleet holds the profile on the same terms as the `others` case

#### Scenario: Enrollment has not reported yet

- **WHEN** the enrollment has a UPN-backed user identity and no observation has been recorded
- **THEN** Fleet holds the profile, because the hold releases on a positive `user` report rather than on the absence of a contrary one

#### Scenario: Held profile delivers after first sign-in

- **WHEN** a held user-scoped profile's enrollment reports user context present in a later session
- **THEN** Fleet enqueues the profile on the next reconcile and the profile proceeds through the normal delivery and verification path

#### Scenario: Host with user context present is not delayed

- **WHEN** a user-scoped profile applies to a host whose enrollment has already reported user context present
- **THEN** Fleet enqueues the profile immediately, with no hold

### Requirement: Failing user-scoped profiles when no user identity is bound

For an enrollment with no bound MDM user identity, where a user-scoped write can never succeed, Fleet SHALL fail the profile with an explanatory detail instead of holding it or resending it. Fleet MUST NOT leave such a profile Pending indefinitely, and MUST NOT resend it on subsequent reconciles.

#### Scenario: User-scoped profile on an enrollment with no user identity

- **WHEN** a user-scoped profile applies to a host whose enrollment has no bound MDM user identity, such as a programmatic fleetd enrollment
- **THEN** the profile reports Failed with a detail stating that user-channel profiles require an enrollment that carries a user identity

#### Scenario: Failure is not retried

- **WHEN** a user-scoped profile has failed because the enrollment has no bound user identity
- **THEN** Fleet does not resend the profile on subsequent reconciles, and the failure does not oscillate between Pending and Failed

### Requirement: Retry accounting for user-context rejections

A rejection of a user-scoped command SHALL NOT consume a profile's retry budget while Fleet is waiting for a user context that can still arrive. In all other states the existing retry accounting applies unchanged. The enrollment's user-context state, not the returned status code, SHALL determine whether the exemption applies, because no status code reliably indicates missing user context: 405 is returned both for a user-scope write before user context initializes and for a CSP node that is unsupported on the device's Windows edition. Fleet MUST also distinguish a rejection from an unrelated atomic rollback by examining the per-LocURI statuses rather than the enclosing `Atomic` status alone.

#### Scenario: Pre-sign-in rejection does not spend a retry

- **WHEN** a user-scoped profile is rejected with a user-context status such as 500 or 405 on its user-scoped LocURI, and the enrollment has a bound user identity with no observed user context
- **THEN** the profile returns to Pending and its retry count is unchanged

#### Scenario: Atomic rollback caused by an already-existing node still retries normally

- **WHEN** a profile's `Atomic` returns 507 because a nested `Add` returned 418 while sibling commands returned 216
- **THEN** Fleet applies its existing already-exists resend path and normal retry accounting, because the nested statuses show this is not a user-context rejection

#### Scenario: Rejection after user context is present spends a retry

- **WHEN** a user-scoped profile is rejected after the enrollment has reported user context present
- **THEN** the existing retry accounting applies and the profile reaches Failed after the configured maximum retries

#### Scenario: Unsupported CSP node is not mistaken for missing user context

- **WHEN** a user-scoped profile targets a node that is unsupported on the device's Windows edition and is rejected with 405 while the enrollment has reported `user`
- **THEN** normal retry accounting applies and the profile reaches Failed, because the exemption is keyed on the enrollment's state rather than on the 405 status

### Requirement: Mixed-scope profiles are held or failed as a unit

A profile containing both device-scoped and user-scoped targets SHALL be treated as user-scoped for delivery purposes. Fleet MUST NOT deliver its device-scoped portion separately.

#### Scenario: Mixed-scope profile held before sign-in

- **WHEN** a profile contains both `./Device/` and `./User/` targets and the enrollment has no observed user context but a bound user identity
- **THEN** the whole profile is held, and none of its commands are enqueued

#### Scenario: Mixed-scope profile delivered whole after sign-in

- **WHEN** a held mixed-scope profile's enrollment reports user context present
- **THEN** the whole profile is delivered in one command, preserving the atomicity its SCEP portion depends on

### Requirement: Device-scoped profile delivery is unchanged

Fleet SHALL deliver device-scoped profiles exactly as it does today, with no dependency on user context.

#### Scenario: Device-scoped profile during OOBE

- **WHEN** a device-scoped profile applies to a host whose enrollment has no observed user context
- **THEN** Fleet enqueues and delivers it immediately, with unchanged status and retry behavior

### Requirement: Documented user-channel support

Fleet's documentation SHALL state which enrollment types support user-channel Windows profiles, that user-scoped profiles remain Pending until a user signs in on enrollments where user context can still arrive, and that they fail with an explanatory reason on enrollments with no bound user identity.

#### Scenario: Administrator reads the custom settings documentation

- **WHEN** an administrator consults the Windows custom settings documentation before assigning a `./User/` profile
- **THEN** the documentation states the enrollment requirement, the Pending-until-sign-in behavior, and the mixed-scope hold rule
