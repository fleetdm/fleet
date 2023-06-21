# macOS updates

## End user macOS update reminders via Nudge

_Available in Fleet Premium_

End users can be reminded and encouraged to update macOS (via [Nudge](https://github.com/macadmins/nudge)).

![Nudge window](https://raw.githubusercontent.com/fleetdm/fleet/main/docs/images/nudge-window.png)

A Fleet admin can set a minimum version and deadline for Fleet-enrolled hosts. If an end user's machine is below the minimum version, the Nudge window above will periodically appear to encourage them to upgrade. The end user has the option to defer the update, but as the deadline approaches, the Nudge window appears more frequently. 

When the end user machine is below the minimum version, Nudge applies the following behavior:

|                                      | > 1 day before deadline | < 1 day before deadline | past deadline         |
| ------------------------------------ | ----------------------- | ----------------------- | --------------------- |
| Nudge window frequency               | Once a day at 8pm GMT   | Once every 2 hours      | Immediately on login  |
| End user can defer                   | ✅                      | ✅                      | ❌                    |
| Nudge window is dismissable          | ✅                      | ✅                      | ❌                    |


### How to set up

To set the macOS updates settings in the UI, visit the **Controls** section and then select the **macOS updates** tab. 

To set the macOS updates settings via CLI, use the configurations listed [here](https://fleetdm.com/docs/using-fleet/configuration-files#mdm-macos-updates).

### Requirements
- Fleet Premium or Ultimate
- [Fleetd](https://fleetdm.com/docs/using-fleet/fleetd) with Fleet Desktop enabled

### End user experience

After the user clicks "update" in the Nudge window, they will be taken to the standard Apple software update screen: 

![Apple software update screen on macOS 12](https://user-images.githubusercontent.com/5359586/228936740-2e8acf2e-6523-4710-9b3f-8243398bd98e.png)

Here, the user would follow Apple's standard two-step process for macOS updates: 
1. Download the macOS update. This occurs in the background and does not interrupt the end user's work.
2. Initiate the update which does prevent the end user from using the host for a time. 

On Intel Macs, Fleet triggers step 1 (downloading the macOS update) programmatically when a new version is available. This way, when the user arrives on the software update screen, they only need to initiate step 2. 

> On Macs with Apple Silicon (e.g. M1), downloading the macOS update may require end user action. Apple doesn't support downloading the update programmatically on Macs with Apple silicon. 

Step 2 (installing the update) always requires end user action.

### Known issue

Sometimes after the end user clicks "update" on the Nudge window, the end user's Mac will say that macOS is up to date when it isn't. This known issue can create a frustrating experience for the end user. Ask the end user to follow the steps below to troubleshoot:

1. From the Apple menu in the top left corner of your screen, select **System Settings** or **System Preferences**.

2. In the search bar, type "Software Update." Select **Software Update**.

3. Type "Command (⌘)-R" to check for updates. If you see an available update, select **Restart Now** to update.

4. If you still don't see an available update, from the Apple menu in the top left corner of your screen, select **Restart...** to restart your Mac.

5. After your Mac restarts, from the Apple menu in the top left corner of your screen, select **System Settings** or **System Preferences**.

6. In the search bar, type "Software Update." Select **Software Update** and select **Restart Now** to update.

## End user macOS update via built-in macOS notifications

Built-in macOS update reminders are available for all Fleet instances. To trigger these reminders, run the ["Schedule an OS update" MDM command](https://developer.apple.com/documentation/devicemanagement/schedule_an_os_update).

<meta name="pageOrderInSection" value="1502">
<meta name="title" value="MDM macOS updates">
