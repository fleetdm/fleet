## ADDED Requirements

### Requirement: A wedged Windows host can be recovered without reinstalling the MSI
Fleet SHALL provide a way to deliver a fresh enroll secret to a Windows host on which fleetd is already installed but has not enrolled, without depending on the fleetd MSI being installed again. The fleetd install command targets a fixed MSI product GUID, so re-sending it to a device that already has fleetd installed is a no-op on the device and cannot carry a new secret.

#### Scenario: Host with fleetd installed but never enrolled
- **WHEN** the fleetd MSI has installed on a Windows host but the host never completed orbit enrollment, and its secret is consumed or no longer valid
- **THEN** an administrator can trigger delivery of a fresh secret
- **AND** the host enrolls without the MSI being reinstalled and without manual intervention on the device

#### Scenario: Re-sending the install command alone does not recover the host
- **WHEN** only the fleetd install command is re-enqueued to a host that already has the MSI installed
- **THEN** the host does not receive a usable new secret through that command
- **AND** the recovery mechanism is required instead

### Requirement: Recovery mints a fresh secret
Triggering recovery SHALL mint a new secret when the host's previous secret has been consumed, and SHALL reuse the existing secret when it is still unconsumed, consistent with the minting rules for the initial delivery.

#### Scenario: Recovery after consumption
- **WHEN** an administrator triggers recovery for a host whose secret has `consumed_at` set
- **THEN** a new secret is minted and delivered
- **AND** the previous secret remains rejected

#### Scenario: Recovery while the secret is still live
- **WHEN** an administrator triggers recovery for a host whose secret is unconsumed
- **THEN** the same secret value is re-delivered and no new row is created

### Requirement: The agent picks up a re-delivered secret
fleetd on Windows SHALL consult the re-deliverable source on startup and adopt a newer secret found there, so a recovered host enrolls without operator action on the device.

#### Scenario: Agent adopts the new secret on restart
- **WHEN** a fresh secret has been delivered to a wedged host and fleetd restarts
- **THEN** fleetd uses the new secret to enroll
- **AND** the previously stored secret is replaced

#### Scenario: Agent keeps working when no new secret is present
- **WHEN** fleetd starts on a host that is already enrolled and no new secret has been delivered
- **THEN** fleetd continues using its existing credentials and does not re-enroll

### Requirement: Only administrators can trigger recovery
Recovery SHALL be an administrator action. End users SHALL NOT be able to trigger delivery of an enroll secret from the My device page, because minting an enrollment credential is an administrator decision.

#### Scenario: End user attempt is refused
- **WHEN** an end user attempts to trigger the recovery action from the My device page
- **THEN** the request is refused with an authorization error

#### Scenario: Administrator action succeeds
- **WHEN** an administrator with the appropriate role triggers recovery for a Windows host
- **THEN** the action is accepted and delivery is scheduled

### Requirement: Recovery does not create unbounded live secrets
Repeated recovery attempts or repeated failed installs SHALL NOT leave an unbounded number of simultaneously valid secrets for one host, and the failure SHALL be visible rather than silent.

#### Scenario: Repeated session alerts do not mint per alert
- **WHEN** a host that looks fleetd-less produces many MDM session-start alerts
- **THEN** the number of unconsumed secrets for that host does not grow with the number of alerts

#### Scenario: Repeated install failure is observable
- **WHEN** the fleetd install repeatedly fails on a host
- **THEN** the condition is visible to an administrator rather than silently retried forever
