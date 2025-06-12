# Android

## Reference links
- [Android Management API](https://developers.google.com/android/management/reference/rest)
- [Google Cloud Pub/Sub API](https://cloud.google.com/pubsub/docs/reference/rest)
- [Google Cloud console pub/sub topics](https://console.cloud.google.com/cloudpubsub/topic/list)

## Setup an Android MDM environment

Create a Google service account with the following Roles
- Android Management User
- Pub/Sub Admin

To do so:
* Follow instructions at https://developers.google.com/android/management/service-account to create the project and service account
* Follow instructions at https://developers.google.com/android/management/notifications to create pub/sub notifications
* Troubleshooting: watch the video of Gabe and Victor discussion post-standup: https://us-65885.app.gong.io/call?id=4731209913082368849 (starting at the 12:50 timestamp)

## Configure dev environment

Using the `credentials.json` of the above account:
```bash
export FLEET_DEV_ANDROID_SERVICE_CREDENTIALS=$(cat credentials.json)
```

Set the feature flag:
```bash
export FLEET_DEV_ANDROID_ENABLED=1
```

To turn on Android MDM, use a Chrome private window (so that you are not logged in with your "fleetdm.com" address). This is only required to enable Android MDM, you can use a normal window for the rest. In "Settings -> Integrations -> MDM -> Turn On Android -> Connect", use a personal email address (not a "fleetdm.com" one). Select "Sign-up for Android only". Domain name is not important ("test.com" for example). No need to fill anything in the "Data protection officer" and "EU representative" sections, just check the checkbox.

## Known issues and limitations
- The Fleet server URL must be public for pub/sub to work properly.
- The Fleet server URL cannot change -- pub/sub is set up with one URL.

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
    Fleet server->>-Admin: UI redirect (new page)

    Admin->>Google: Enterprise signup
    activate Google
    Google->>Fleet server: Signup callback
    deactivate Google
    activate Fleet server
    Fleet server->>+fleetdm.com: Create enterprise/policy/pubsub
    fleetdm.com->>+Google: Create enterprise/policy/pubsub
    Google-->>-fleetdm.com: Created
    fleetdm.com-->>-Fleet server: Created
    Fleet server->>Admin: Self-closing HTML page
    Fleet server--)Admin: Android enabled (SSE)
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

```mermaid
---
title: Partial class diagram
config:
  class:
    hideEmptyMembersBox: true
---
classDiagram
    direction LR
    class `android.Service`
    <<interface>> `android.Service`
    class `android/service.Service`
    `android/service.Service` ..|> `android.Service`: implements

    class `fleet.AndroidDatastore`
    <<interface>> `fleet.AndroidDatastore`
    class `fleet.Datastore`
    <<interface>> `fleet.Datastore`
    class `android.Datastore`
    <<interface>> `android.Datastore`
    `android/service.Service` *-- `fleet.AndroidDatastore`: uses
    `fleet.Datastore` *-- `fleet.AndroidDatastore`: contains
    `mysql.Datastore` ..|> `fleet.Datastore`: implements
    `fleet.AndroidDatastore` *-- `android.Datastore`: contains
    `mysql.Datastore` *-- `android.Datastore`: contains
    `android/mysql.Datastore` ..|> `android.Datastore`: implements
```

## Security and authentication

Android enterprise signup callback is authenticated by a token in the callback URL. The token is created by Fleet server.

Getting the Android device enrollment token is authenticated with the Fleet enroll secret.

Pub/sub push callback is authenticated by a `token` query parameter. This token is created by Fleet server. As of March 2025, this token cannot be easily rotated. We could add another level of authentication where the Fleet server would need to check with Google to authenticate the pub/sub message:
- [Authentication for push subscriptions](https://cloud.google.com/pubsub/docs/authenticate-push-subscriptions)
