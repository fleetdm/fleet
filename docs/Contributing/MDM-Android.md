# Android

## Reference links
- [Android Management API](https://developers.google.com/android/management/reference/rest)
- [Google Cloud Pub/Sub API](https://cloud.google.com/pubsub/docs/reference/rest)
- [Google Cloud console pub/sub topics](https://console.cloud.google.com/cloudpubsub/topic/list)

## Configure dev environment

Create a Google service account with the following Roles
- Android Management User
- Pub/Sub Admin

Using the `credentials.json` of the above account:
```bash
export FLEET_DEV_ANDROID_SERVICE_CREDENTIALS=$(cat credentials.json)
```

Set the feature flag:
```bash
export FLEET_DEV_ANDROID_ENABLED=1
```

Note: The Fleet server URL must be public for pub/sub to work properly.

## Architecture diagrams

```mermaid
---
title: Enable Android MDM
---
sequenceDiagram
    autonumber
    actor Admin
    participant Fleet server
    participant fleetdm.com
    participant Google

    Admin->>+Fleet server: Enable Android
    Fleet server->>+fleetdm.com: Get signup url
    fleetdm.com->>+Google: Get signup url
    Google-->>-fleetdm.com: Signup url
    fleetdm.com-->>-Fleet server: Signup url
    Fleet server->>-Admin: UI redirect

    Admin->>Google: Enterprise signup
    activate Google
    Google->>Fleet server: Signup callback (self-closing HTML page)
    deactivate Google
    activate Fleet server
    Fleet server->>+fleetdm.com: Create enterprise
    fleetdm.com->>+Google: Create enterprise and pub/sub
    Google-->>-fleetdm.com: Created
    fleetdm.com-->>-Fleet server: Created
    Fleet server->>Admin: Android enabled (SSE)
    deactivate Fleet server
```

```mermaid
---
title: Enroll BYOD Android device
---
sequenceDiagram
    autonumber
    actor Admin
    actor Employee
    participant Enroll page
    participant Fleet server
    participant fleetdm.com
    participant Google

    Admin->>+Fleet server: Get signup link
    Fleet server-->>-Admin: Signup link

    Admin->>Employee: Email signup link
    Employee->>+Fleet server: Click signup link
    Fleet server-->>-Enroll page: HTML page
    Employee->>+Enroll page: Click enroll
    Enroll page->>+Fleet server: Get enroll token
    Fleet server->>+fleetdm.com: Get enroll token
    fleetdm.com->>+Google: Get enroll token
    Google-->>-fleetdm.com: Enroll token
    fleetdm.com-->>-Fleet server: Enroll token
    Fleet server-->>-Enroll page: Enroll token
    Enroll page->>-Employee: Redirect to enroll flow

    Employee->>+Google: Enroll device
    Google-->>Employee: Device enrolled
    Google--)Fleet server: Pub/Sub push: ENROLLMENT
    Google--)-Fleet server: Pub/Sub push: STATUS_REPORT

    Admin->>+Fleet server: Get hosts
    Fleet server-->>-Admin: Hosts (including Android)
```

## Security and authentication

Android enterprise signup callback is authenticated by a token in the callback URL. The token is created by Fleet server.

Getting the Android device enrollment token is authenticated with the Fleet enroll secret.

Pub/sub push callback is authenticated by a `token` query parameter. This token is created by Fleet server. As of March 2025, this token cannot be easily rotated. We could add another level of authentication where the Fleet server would need to check with Google to authenticate the pub/sub message:
- [Authentication for push subscriptions](https://cloud.google.com/pubsub/docs/authenticate-push-subscriptions)
