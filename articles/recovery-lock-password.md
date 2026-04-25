# Recovery lock password

_Available in Fleet Premium_

Fleet can set a recovery lock password on Apple Silicon Macs enrolled in Fleet MDM. This password lets IT admins unlock a device at the recoveryOS screen if the end user forgets their local password or the device needs to be recovered.

Fleet automatically generates, encrypts, and stores the password server-side. Admins can view or rotate it from the Fleet UI or API.

## Prerequisites

- macOS host with Apple Silicon (ARM)
- Host enrolled in Fleet MDM
- Fleet Premium license

## Enable recovery lock password

### UI

1. Go to **Controls > OS settings > Passwords**.
2. Select your desired fleet from the dropdown
3. Check **Turn on Recovery Lock password**.
4. Click **Save**

Fleet will begin setting passwords on eligible hosts automatically in the current fleet. Progress appears in each host's **OS settings** status.

### fleetctl

Add `enable_recovery_lock_password: true` under the `mdm` key in your fleet or unassigned (global) YAML config:

```yaml
mdm:
  enable_recovery_lock_password: true
```

Then apply:

```sh
fleetctl apply -f config.yml
```

### API

For unassigned hosts:
```
PATCH /api/latest/fleet/config
```

```json
{
  "mdm": {
    "enable_recovery_lock_password": true
  }
}
```

For a specific fleet:

```
PATCH /api/latest/fleet/fleets/{fleet_id}
```

```json
{
  "config": {
    "mdm": {
      "enable_recovery_lock_password": true
    }
  }
}
```

## View the password

1. Go to the **Host details** page for a macOS host.
2. Click **Actions > Show recovery lock password**.
3. The password is displayed in the modal.

This action is logged as a activities visible on the host's and the global activity feed.

### API

Use the [Get host's Recovery Lock password](https://fleetdm.com/docs/rest-api/rest-api#get-hosts-recovery-lock-password) endpoint.


## Rotate the password

Rotation generates a new password and pushes it to the device via an MDM command.

1. Go to the **Host details** page for a macOS host.

Then either: 

2. Click **Actions > Show recovery lock password**.
3. Click **Rotate password**.

or:

2. Click on the **OS settings** indicator in the host summary card.
3. Hover over the Recovery Lock password row
4. Click "Rotate"


Requires maintainer role or higher.

### API

Use the [Rotate host's Recovery Lock password API](https://fleetdm.com/docs/rest-api/rest-api#rotate-hosts-recovery-lock-password) endpoint.


## Status tracking

Recovery lock password status appears alongside other OS settings on the host details page. Possible statuses:

| Status | Meaning |
| --- | --- |
| Verified | Fleet set a recovery lock password for the host. |
| Enforcing (pending) | Fleet is setting a recovery lock password for the host. |
| Removing enforcement (pending) | Fleet is unsetting the recovery lock password for the host. |
| Failed | Fleet failed to set a recovery lock password for the host.|

## Disable recovery lock password

Turn off the setting using the same path as enabling (UI, fleetctl, or API). Fleet will send a clear command to remove the password from enrolled hosts.

## How it works

- **Password format**: 6 groups of 4 alphanumeric characters separated by dashes (e.g., `A3B7-C9D2-E5F8-G4H6-J2K9-L7M3`). Characters that look similar (0/O, 1/I/l) are excluded for readability.
- **Encryption**: Passwords are encrypted with AES-256 using the server's private key before storage. They are never stored in plaintext.
- **Secret injection**: Passwords are injected into MDM commands at delivery time using placeholder expansion, so plaintext passwords never appear in the command queue.
- **Activities**: Fleet logs activities when a password is set, rotated, or viewed, and when the
  feature is enabled or disabled for a fleet. These appear on the host's activity timeline and in the global
  activity feed.

<meta name="articleTitle" value="Recovery lock password">
<meta name="authorFullName" value="Jacob Shandling">
<meta name="authorGitHubUsername" value="jacobshandling">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-03-13">
<meta name="description" value="Set, view, and rotate recovery lock passwords on Apple Silicon Macs with Fleet MDM.">
