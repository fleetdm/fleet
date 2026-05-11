# Android Managed App Configurations

## [Allow Work Profile Widgets](allow-work-profile-widgets-for-APPNAME.json)

- Change `APPNAME` in the filename and the `"APPNAME Widget": "Allow Work Profile Widgets for APPNAME"` line in the file to the name of the app you'd like to allow Work Profile widgets for.
- Scope this configuration to the app:
  - UI: Software > Select the app > Actions > Edit configuration
  - [GitOps](https://fleetdm.com/docs/configuration/yaml-files#app-store-apps): Under the `app_store_app` entry, add the path under the `configuration.path` for the app.
- No need to add the identifier (i.e., `com.google.android.calendar` for Google Calendar) to the configuration. Fleet automatically takes care of this, since the config is scoped to the app.
- [Learn more](https://fleetdm.com/guides/install-app-store-apps#configuration) about app configurations.
