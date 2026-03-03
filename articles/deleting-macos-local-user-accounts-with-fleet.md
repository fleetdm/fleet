# Deleting macOS local user accounts with Fleet

The recommended method for deleting local user accounts on macOS is to use `dscl`, such as `dscl . -delete /Users/testing`, but if you run it in a script from Fleet, you'll encounter an error:

```
<main> delete status: eDSPermissionError
<dscl_cmd> DS Error: -14120 (eDSPermissionError)

script execution error: exit status 40
```

An alternative way to remove accounts is to use `sysadminctl`, such as `sysadminctl -deleteUser testing -secure`, but you'll run into this error:

```
2026-02-17 16:30:59.669 sysadminctl[1488:28502] Killing all processes for UID 502
2026-02-17 16:30:59.671 sysadminctl[1488:28502] Securely removing testing's home at /Users/testing
2026-02-17 16:30:59.701 sysadminctl[1488:28502] Deleting Public share point for testing
2026-02-17 16:30:59.724 sysadminctl[1488:28502] Deleting record for testing
2026-02-17 16:30:59.730 sysadminctl[1488:28502] AOSKit INFO: Disabling BTMM for user, no zone found for uid=502, usersToZones: (null)
2026-02-17 16:30:59.784 sysadminctl[1488:28502] ### Error:-14120 File:/AppleInternal/Library/BuildRoots/4~B5FeugCYEZd96X3zqIDDgARflf-BD_RFQt1Gd-I/Library/Caches/com.apple.xbs/Sources/Admin/DSRecord.m Line:563
```

This deletes the user's home folder, but the account isn't removed. This can be confirmed in the Users & Groups pane of System Settings.

The reason for this is that `fleetd`, the Fleet agent that runs the scripts, requires full disk access permissions to run these commands to delete users. Grant full disk access to `fleetd` using our [configuration profile](https://github.com/fleetdm/fleet/blob/main/docs/solutions/macos/configuration-profiles/allow-fleetd-full-disk-access.mobileconfig).

Once that profile has been installed on the device, `dscl . -delete` and `sysadminctl -deleteUser` will work. `dscl` won't show any output, but `sysadminctl` output will look like this:

```
2026-02-17 16:33:59.516 sysadminctl[1558:31022] Killing all processes for UID 502
2026-02-17 16:33:59.518 sysadminctl[1558:31022] Securely removing testing's home at /Users/testing
2026-02-17 16:33:59.562 sysadminctl[1558:31022] Deleting Public share point for testing
2026-02-17 16:33:59.602 sysadminctl[1558:31022] Deleting record for testing
2026-02-17 16:33:59.606 sysadminctl[1558:31022] AOSKit INFO: Disabling BTMM for user, no zone found for uid=502, usersToZones: (null)
```


<meta name="category" value="guides">
<meta name="authorFullName" value="Steven Palmesano">
<meta name="authorGitHubUsername" value="spalmesano0">
<meta name="publishedOn" value="2026-03-03">
<meta name="articleTitle" value="Deleting macOS local user accounts with Fleet">
