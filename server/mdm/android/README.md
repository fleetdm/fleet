The MDM Android package attempts to decouple Android-specific service and datastore implementations from the core Fleet server code.

Any tightly coupled code that needs both the core Fleet server and the Android-specific features must live in the main server/fleet,
server/service, and server/datastore packages. Typical example are MySQL queries. Any code that implements Android-specific functionality
should live in the server/mdm/android package. For example, the common code from server/datastore package can call the android datastore
methods as needed.

This decoupled approach attempts to achieve the following goals:
- Easier to understand and find Android-specific code.
- Easier to fix Android-specific bugs and add new features.
- Easier to maintain Android-specific feature branches.
- Faster Android-specific tests, including ability to run all tests in parallel.

## Setup an Android MDM environment

* Follow instructions at https://developers.google.com/android/management/service-account to create the project and service account
* Follow instructions at https://developers.google.com/android/management/notifications to create pub/sub notifications
* Troubleshooting: watch the video of Gabe and Victor discussion post-standup: https://us-65885.app.gong.io/call?id=4731209913082368849

Start fleet with `FLEET_DEV_ANDROID_ENABLED=1` (enable the feature flag) and `FLEET_DEV_ANDROID_SERVICE_CREDENTIALS=$(cat path/to/your/service-account.json)` (provide the service account credentials).

Use a Chrome private window to enable Android MDM (so that you are not logged in with the fleetdm.com address). This is only required to enable Android MDM, you can use a normal window for the rest. In "Settings -> Integrations -> MDM -> Turn On Android -> Connect", use a personal email address (not a fleetdm.com one). Select "Sign-up for Android only". Domain name is not important ("test.com" for example). No need to fill anything in the "Data protection officer" and "EU representative" sections, just check the checkbox.
