# Patch notifications without the Fleet Desktop Dock app


## Introduction

On macOS, **Notify before patching** warns end users an hour before Fleet closes and updates an open app. The notification comes from the [Fleet Desktop app](https://fleetdm.com/software-catalog/fleet-desktop-darwin), the version of Fleet Desktop that lives in the **Applications** folder and the Dock. The script Fleet sends to show the notification runs that app from `/Applications/Fleet Desktop.app`. If it's missing, Fleet doesn't notify the end user and skips the patch.

That's a problem for two kinds of teams:

- Teams that want to keep the classic menu bar icon. End users already go to the Fleet Desktop menu bar icon for **My device** and **Self-service**. A second Fleet Desktop in the Dock and **Applications** folder gives them two entry points with the same name.
- Teams with their own branded desktop app. If you've built your own app on top of Fleet Desktop, with your company's name and icon, Fleet's notification script won't use it. It only runs the Fleet Desktop app, so without that app you can't use **Notify before patching**.

You don't have to choose. The Fleet Desktop app has its own bundle ID, so it runs side by side with the menu bar icon or your branded app. Install it, hide it with a post-install script, and keep the app your end users already know. They see a notification before Fleet updates an app they have open, and nothing else changes.

This guide covers how to:

1. Hide the Fleet Desktop app after every install and update.
2. Install the app without showing it, or its install policy, to end users.
3. Turn on **Notify before patching** for the apps you choose.
4. Keep the app up to date.


## What you need

- Fleet Premium. To hide the install policy from end users, you need Fleet 4.94 or later.
- Macs enrolled with Fleet's agent (fleetd).
- The Fleet Desktop app 1.5.0 or later. Fleet-maintained apps install the latest version.


## Two things called Fleet Desktop

| | Fleet Desktop menu bar icon | Fleet Desktop app |
|:--|:--|:--|
| Bundle ID | `com.fleetdm.desktop` | `com.fleetdm.fleet-desktop` |
| Comes from | Fleet's agent (fleetd) | A [Fleet-maintained app](https://fleetdm.com/software-catalog/fleet-desktop-darwin) |
| Lives in | `/opt/orbit/bin/desktop/` | `/Applications/Fleet Desktop.app` |
| Shows patch notifications | No | Yes |

A branded app is a third app with its own bundle ID. Fleet doesn't look for it when it sends a notification.

This guide keeps the menu bar icon, or your branded app, as the end user's entry point and uses the Fleet Desktop app only for notifications.


## Step 1: Hide the app after every install

Hiding the app in Finder takes one command: `chflags hidden`. Run it as the app's post-install script, so it runs again every time Fleet installs or updates the app. Each update replaces the app bundle, and the new bundle isn't hidden until the script runs.

```zsh
#!/bin/zsh
# Post-install script for the Fleet Desktop app (fleet-desktop/darwin).
# Hides the app in Finder so end users keep using the menu bar icon or your branded app.
APP="/Applications/Fleet Desktop.app"

if [[ ! -d "$APP" ]]; then
  echo "Fleet Desktop app not present, nothing to hide"
  exit 0
fi

/usr/bin/chflags hidden "$APP" || { echo "chflags failed"; exit 1; }
echo "Hidden: $(/bin/ls -lOd "$APP")"
exit 0
```

`chflags hidden` only hides the app in Finder. Fleet's notification script runs the app by its path, so a hidden app still shows notifications. If the post-install script fails, Fleet marks the install as failed, and you'll see it in the host's activity feed.


## Step 2: Add the Fleet Desktop app


### In the Fleet UI

1. Go to **Software**, select a fleet, and select **Add software**.
2. Select **Add** next to **Fleet Desktop**.
3. Select **Automatic**. Fleet creates a policy that installs the app on Macs that don't have it, and reinstalls it if an end user deletes it.
4. Select **Advanced options** and paste the script from Step 1 as the post-install script.
5. Select **Add software**.

Don't make the app available in **Self-service**. End users already have the menu bar icon or your branded app, and a second Fleet Desktop on the **Self-service** page only confuses them.

To install the app on new Macs before the end user reaches the desktop, also add it to [setup experience](https://fleetdm.com/guides/setup-experience#install-software).


### With GitOps

Add the app to the fleet's `software.fleet_maintained_apps`. Leave out `self_service`, and set `categories` to an empty list so the catalog's default category isn't applied:

```yaml
software:
  fleet_maintained_apps:
    - slug: fleet-desktop/darwin
      categories: []
      post_install_script:
        path: ../lib/macos/scripts/hide-fleet-desktop-app.sh
```

Then add a policy that installs it. Set `hidden: true` (Fleet 4.94 or later) so end users don't see the policy on **My device**. There's nothing for them to do, and a failing policy they can't fix only causes support tickets:

```yaml
policies:
  - name: "[Install] Fleet Desktop app"
    platform: darwin
    description: "Checks that the Fleet Desktop app is installed. Fleet needs it to notify end users before it updates an open app."
    resolution: "Fleet installs this automatically. No action needed."
    query: "SELECT 1 FROM apps WHERE bundle_identifier = 'com.fleetdm.fleet-desktop';"
    hidden: true
    install_software:
      fleet_maintained_app_slug: fleet-desktop/darwin
```

`install_software` only works on a fleet that also lists `fleet-desktop/darwin` under `software.fleet_maintained_apps`. Reference this policy with its own `path:` in each fleet that gets the app, rather than through a `paths:` glob shared with fleets that don't.


## Step 3: Turn on Notify before patching

Choose a few apps to start with. Menu bar apps that are almost always running are good first candidates: they're open when the patch is due, so you see the notification quickly.

In the Fleet UI:

1. Go to **Software**, select a fleet, and open the app's details page.
2. Select **Actions > Deploy**.
3. Check **Patch**, then select **Force patch > Notify before patching**.

With GitOps, set `notify_before_patching: true` on the app's patch policy. It replaces `patch_when_closed`, so use one or the other:

```yaml
- name: "[Updates] Spotify"
  type: patch
  fleet_maintained_app_slug: spotify/darwin
  install_software: true
  notify_before_patching: true
  description: "Checks that Spotify is up to date if installed."
  resolution: "Fleet updates Spotify automatically and notifies you an hour before if it's open."
```

To learn what end users see, and how the 1-hour timer and the 5-minute reminder work, see [Patching end user experience](https://fleetdm.com/guides/patching-end-user-experience#force-patch-notify-before-patching).


## Step 4: Keep the app up to date

Fleet downloads each new version of the Fleet Desktop app, but Macs that already have the app only get it the next time it's installed. To [keep it up to date](https://fleetdm.com/guides/fleet-desktop#update-fleet-desktop), add a patch policy for `fleet-desktop/darwin` with **Patch when app is closed**. Because the app is hidden, end users rarely open it, so it's almost always closed when the patch is due. Each update runs the post-install script again, so the app stays hidden.


## Check that it works

On a test Mac in the fleet:

1. Run `ls -lOd "/Applications/Fleet Desktop.app"` in Terminal and look for `hidden` in the output. Finder no longer shows the app in **Applications**.
2. Open an app you switched to **Notify before patching** while it's out of date. When its patch policy fails, the notification appears. Select **Update now**, or wait for the update an hour later.
3. In Fleet, go to **Hosts**, select the host, and check the activity feed to confirm the update installed.

If you see a "failed to notify" activity, Fleet couldn't run the Fleet Desktop app on that host. Check that the install policy passes, and that the app is at `/Applications/Fleet Desktop.app`.


## Good to know

- Don't move or rename the app. Fleet's notification script only looks for it at `/Applications/Fleet Desktop.app`.
- Hidden isn't removed. An end user who knows where to look can still open the app, for example with **Go > Go to Folder** in Finder.
- This setup doesn't change the Fleet Desktop menu bar icon. If you [hide the menu bar icon](https://fleetdm.com/guides/fleet-desktop#hide-the-menu-bar-icon-on-macos), for example because end users have your branded app instead, notifications still work.
- Your branded app keeps working. The Fleet Desktop app doesn't replace or change it.
- If the screen is locked or another notification is showing, nothing installs, and Fleet tries again on the next policy run. See [When notifications aren't shown](https://fleetdm.com/guides/patching-end-user-experience#when-notifications-arent-shown).
- Self-service and setup experience installs don't notify. Fleet only notifies when a patch policy installs the update.


## Conclusion

You don't need the Fleet Desktop app in front of end users to use **Notify before patching**. With a short post-install script and a hidden policy, you get patch notifications on macOS and keep the classic menu bar icon or your own branded app. Start with a few apps, check the activity feed, then roll **Notify before patching** out to the apps where a forced quit would cost end users their work.

Want to learn more about patching in Fleet? Check out [Patching end user experience](https://fleetdm.com/guides/patching-end-user-experience) and [Fleet Desktop](https://fleetdm.com/guides/fleet-desktop), or the other [guides for macOS, Windows, and Linux](https://fleetdm.com/guides).

<meta name="articleTitle" value="Patch notifications without the Fleet Desktop Dock app">
<meta name="authorFullName" value="Adam Baali">
<meta name="authorGitHubUsername" value="AdamBaali">
<meta name="category" value="guides">
<meta name="publishedOn" value="2026-11-20">
<meta name="description" value="Use Notify before patching with the classic menu bar icon or your own branded app by hiding the Fleet Desktop app with a post-install script.">
