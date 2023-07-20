# Custom macOS settings

In Fleet you can enforce custom settings on your macOS hosts using configuration profiles.

## Enforce custom settings

To enforce custom settings, we will do the following steps:

1. Create a configuration profile with iMazing Profile editor
2. Upload the profiles to Fleet
3. Confirm the setting is enforced

### Step 1: create a configuration profile

How to create a configuration profile with iMazing Profile Creator:

1. Download and install [iMazing Profile Creator](https://imazing.com/profile-editor).

2. Open iMazing Profile Creator and select macOS in the top bar.

3. Find and choose the settings you'd like to enforce on your macOS hosts. Fleet recommends limiting the scope of the settings a single profile: only include settings from one tab in iMazing Profile Creator (ex. **Restrictions** tab). To enforce more settings, you can create and add additional profiles.

4. In iMazing Profile Creator, select the **General** tab. Enter a descriptive name in the **Name** field. When you add this profile to Fleet, Fleet will display this name in the Fleet UI.

5. In your top menu bar select **File** > **Save As...** and save your configuration profile. Make sure the file is saved as .mobileconfig.

### Step 2: upload configuration profile to Fleet

In Fleet, you can upload configuration profiles using the Fleet UI or fleetctl command-line tool.

The Fleet UI method is a good start if you're just getting familiar with Fleet.

The fleetctl CLI method enables managing configuration profiles in a Git repository. This way you can enforce code review and benefit from Git's change history.

Fleet UI:

1. In the Fleet UI, head to the **Controls > macOS settings > Custom settings** page.

2. Choose which team you want to add the configuration profile to by selecting the desired team in the teams dropdown in the upper left corner. Teams are available in Fleet Premium.

3. Select **Upload** and choose your configuration profile. After your configuration profile is uploaded to Fleet, Fleet will apply the profile to all macOS hosts in the selected team. Thereafter, the profile will be applied to new macOS hosts that enroll to that team.

fleetctl CLI:

1. Choose which team you want to add the configuration profile to.

In this example, we'll add a configuration profile to the "Workstations (canary)" team so that the setting only gets enforced on hosts assigned to this team.

2. Create a `workstations-canary-config.yaml` file:

```yaml
apiVersion: v1
kind: team
spec:
  team:
    name: Workstations (canary)
    mdm:
      macos_settings:
        custom_settings:
          - /path/to/configuration_profile.mobileconfig
    ...
```

Learn more about team configurations options [here](./configuration-files/README.md#teams).

To enforce settings on hosts that aren't assigned to a team ("No team"), we'll need to create an `fleet-config.yaml` file:

```yaml
apiVersion: v1
kind: config
spec:
  mdm:
    macos_settings:
      custom_settings:
        - /path/to/configuration_profile.mobileconfig
  ...
```

Learn more about configuration options for hosts that aren't assigned to a team [here](./configuration-files/README.md#organization-settings).

3. Add an `mdm.macos_settings.custom_settings` key to your YAML document. This key accepts an array of paths to your configuration profiles.

4. Run the `fleetctl apply -f workstations-canary-config.yml` command to upload the configuration profiles to Fleet. Note that this will override any configuration profiles added using the Fleet UI method.

### Step 3: confirm the setting is enforced

1. In the Fleet UI, head to the **Controls > macOS settings** tab.

2. In the top box, with "Verified," "Verifying," "Pending," and "Failed" statuses, click each status to view a list of hosts:

* Verified: hosts that installed all configuration profiles. Fleet has verified with osquery.

* Latest: hosts that have acknowledged all MDM commands to install configuration profiles. Fleet is verifying the profiles are installed with osquery.

* Verifying: hosts that will receive MDM commands to install configuration profiles when the hosts come online.

* Failed: hosts that failed to install configuration profiles.

3. In the list of hosts, click on an individual host and click the **macOS settings** item to see the status for a specific setting.


<meta name="pageOrderInSection" value="1504">
<meta name="title" value="MDM custom macOS settings">
<meta name="description" value="Learn how to enforce custom settings on macOS hosts using Fleet's configuration profiles.">
<meta name="navSection" value="Device management">