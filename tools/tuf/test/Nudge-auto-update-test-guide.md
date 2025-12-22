# Nudge auto-update test guide

## Setup

To test Nudge auto-updates, first setup your local environment for testing Orbit. This guide assumes
a macOS device set up according to the [README](https://github.com/fleetdm/fleet/main/tools/tuf/test/README.md), where we'll run most of the 
commands, TUF server, Orbit and the Fleet server.

## Add Nudge to TUF repo

The process for adding Nudge to the TUF repo is similar to the process for adding new versions of
Orbit, osqueryd, or Fleet Desktop.

```sh
# Generate nudge app bundle.
make nudge-app-tar-gz version=1.1.10.81462 out-path=.

# Push the nudge target as a new version
./tools/tuf/test/push_target.sh macos nudge nudge.app.tar.gz 1.1.10.81462
```

## Verify Nudge installation locally

Confirm that your Fleet server settings for `app_config.mdm.macos_updates.minimum_version` and
`app_config.mdm.macos_updates.deadline` are empty and restart the Fleet server if necessary.

Run the `fleet-osquery` installer generated for your local environment. After Orbit has launched,
wait for Orbit to perform any necessary updates for Orbit, osqueryd, or Fleet Deskotp. You can find
the locally installed files at `opt/orbit/bin/`.

At this point, Nudge should not be installed. Try launching Nudge in demo mode to confirm.

```sh
opt/orbit/bin/nudge/macos/stable/Nudge.app/Contents/MacOS/Nudge -demo-mode
```

Next, edit your Fleet server configuration to set `app_config.mdm.macos_updates.minimum_version` and
`app_config.mdm.macos_updates.deadline`. The specific settings don't matter here so long as they are
valid (i.e. mimumum version follows the "13.0.1" pattern and deadline follows the "2023-12-31" pattern).

At the next update interval, Orbit should download and install Nudge. After 30-60 seconds, try again
to launch Nudge in demo mode and check the version by clicking the info icon in the upper left
corner of the Nudge window. 

## Logs 

Orbit logs can be found locally at `private/var/log/orbit`.

## Troubleshooting

### Trouble generating nudge-app-tar-gz

The `make nudge-app-tar-gz` script was written to be compatible with the current Nudge release as of
the time of this writing (version 1.1.10.81462). Note that this script may require modification to
work with prior releases of Nudge that use different paths for the package. In such cases, the
script will raise an error stating that a file was not found at the expected path. 

For example, the script needs "Applications/Utilities" to be added certain paths as shown below in
order for it to work with version 1.1.7.81411. 

```
	$(TMP_DIR)/nudge_pkg_payload_expanded/Applications/Utilities/Nudge.app/Contents/MacOS/Nudge --version
	tar czf $(out-path)/nudge.app.tar.gz -C $(TMP_DIR)/nudge_pkg_payload_expanded/Applications/Utilities/ Nudge.app
```

The change above was observed to work with several other prior versions tested as well. In other
cases, you may need to inspect the directory structure of the expanded package and tweak the script
accordingly. 




