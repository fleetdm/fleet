# OS updates

_Available in Fleet Premium_

In Fleet you can enforce OS updates on your macOS and Windows hosts remotely.

On macOS, OS updates are enforced via Nudge. Windows

## Enforce OS updates

You can enforce OS updates in the Fleet UI, with the Fleet API, or with the command-line interface (CLI).

Fleet UI:

1. In Fleet, head to the **Controls** > **OS updates** tab.

2. To enforce OS updates for macOS, select **macOS** and set a **Minimum version** and **Deadline**.

3. For Windows, select **Windows** and set a **Deadline** and **Grace period**.

Fleet API: API documentation is [here](https://fleetdm.com/docs/rest-api/rest-api#modify-team).

fleetctl CLI: Run the `fleetctl apply` command using the `mdm.macos_updates` and `mdm.windows_updates` YAML configuration documented [here](https://fleetdm.com/docs/using-fleet/configuration-files#TODO).

## End user experience

### macOS

End users are be reminded and encouraged to update macOS (via [Nudge](https://github.com/macadmins/nudge)).

![Nudge window](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/nudge-window.png)

When the user selects **Update**, their Mac opens **System Settings > General > Software Update**.

When the end user machine is below the minimum version, Nudge applies the following behavior:

|                                      | > 1 day before deadline | < 1 day before deadline | past deadline         |
| ------------------------------------ | ----------------------- | ----------------------- | --------------------- |
| Nudge window frequency               | Once a day at 8pm GMT   | Once every 2 hours      | Immediately on login  |
| End user can defer                   | ✅                      | ✅                      | ❌                    |
| Nudge window is dismissable          | ✅                      | ✅                      | ❌                    |

<meta name="pageOrderInSection" value="1503">
<meta name="title" value="macOS updates">
<meta name="description" value="Learn how to manage OS updates on macOS and Windows devices.">
<meta name="navSection" value="Device management">
