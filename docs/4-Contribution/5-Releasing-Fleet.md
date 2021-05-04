# Releasing Fleet

1. Update the [CHANGELOG](../../CHANGELOG.md) with the changes that have been made since the last Fleet release. Update the NPM [package.json](../../tools/fleetctl-npm/package.json) with the new version number (do not yet `npm publish`). Update the [Helm chart](../../charts/fleet/Chart.yaml) and [values file](../../charts/fleet/values.yaml) with the new version number.

Commit these changes via Pull Request and pull the changes on the `master` branch locally. Check that `HEAD` of the `master` branch points to the commit with these changes.

2. Tag and push the new release in Git:

``` shell
git tag <VERSION>
git push origin <VERSION>
```

3. Build the new binary bundle (ensure working tree is clean because this will effect the version string built into the binary):

``` shell
make binary-bundle
```

Make note of the SHA256 checksum output at the end of this build command to paste into the release documentation on GitHub.

4. Create a new release on the [GitHub releases page](https://github.com/fleetdm/fleet/releases). Select the newly pushed tag (GitHub should say "Existing tag"). Use the version number as the release title. Use the below template for the release description (replace items in <> with the appropriate values):

````
### Changes

<COPY FROM CHANGELOG>

### Upgrading

Please visit our [update guide](https://github.com/fleetdm/fleet/blob/master/docs/1-Using-Fleet/7-Updating-Fleet.md) for upgrade instructions.

### Documentation

Documentation for this release can be found at https://github.com/fleetdm/fleet/blob/<VERSION>/docs/README.md

### Binary Checksum

**SHA256**
```
<HASH VALUE>  fleet.zip
<HASH VALUE>  fleetctl.exe.zip
<HASH VALUE>  fleetctl-linux.tar.gz
<HASH VALUE>  fleetctl-macos.tar.gz
<HASH VALUE>  fleetctl-windows.tar.gz
```

````

Upload `fleet.zip`, `fleetctl-*.tar.gz`, and `fleetctl.exe.zip`. Click "Publish Release".

5. Push the new version to Docker Hub (ensure working tree is clean because this will effect the version string built into the binary):

``` shell
make docker-push-release
```

6. Publish the new version of `fleetctl` on NPM. Run `npm publish` in the [fleetctl-npm](../../tools/fleetctl-npm/) directory. Note that NPM does not allow replacing a package without creating a new version number. Take care to get things correct before running `npm publish`!

7. Announce the release in the #fleet channel of [osquery Slack](https://osquery.slack.com/join/shared_invite/zt-h29zm0gk-s2DBtGUTW4CFel0f0IjTEw#/) and update the channel's topic with the link to this release. Using `@here` requires admin permissions, so typically this announcement will be done by `@zwass`.

Announce the release via blog post (on Medium) and Twitter (linking to blog post).

8. Crack open a beer and wonder why we haven't yet automated this process. Cheers!

