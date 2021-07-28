# Releasing Fleet

Note: Please prefix versions with `v` (eg. `v4.0.0`) in git tags, Helm charts, and NPM configs.

1. Update the [CHANGELOG](../../CHANGELOG.md) with the changes that have been made since the last Fleet release. Update the NPM [package.json](../../tools/fleetctl-npm/package.json) with the new version number (do not yet `npm publish`). Update the [Helm chart](../../charts/fleet/Chart.yaml) and [values file](../../charts/fleet/values.yaml) with the new version number. Remove all files from the `/changes` top-level directory except for the `.keep` file.

Commit these changes via Pull Request and pull the changes on the `main` branch locally. Check that `HEAD` of the `main` branch points to the commit with these changes.

2. Tag and push the new release in Git:

```shell
git tag v<VERSION>
git push origin v<VERSION>
```

Note that `origin` may be `upstream` depending on your `git remote` configuration. The intent here is to push the new tag to the `github.com/fleetdm/fleet` repository.

GitHub Actions will automatically begin building the new release after the tag is pushed.

---

Wait while GitHub Actions creates and uploads the artifacts...

---

When the Actions Workflow has completed:

3. Edit the draft release on the [GitHub releases page](https://github.com/fleetdm/fleet/releases). Use the version number as the release title. Use the below template for the release description (replace items in <> with the appropriate values):

````
### Changes

<COPY FROM CHANGELOG>

### Upgrading

Please visit our [update guide](https://github.com/fleetdm/fleet/blob/main/docs/1-Using-Fleet/8-Updating-Fleet.md) for upgrade instructions.

### Documentation

Documentation for this release can be found at https://github.com/fleetdm/fleet/blob/<VERSION>/docs/README.md

### Binary Checksum

**SHA256**
```
<COPY FROM checksums.txt>
```
````

When editing is complete, publish the release.

4. Publish the new version of `fleetctl` on NPM. Run `npm publish` in the [fleetctl-npm](../../tools/fleetctl-npm/) directory. Note that NPM does not allow replacing a package without creating a new version number. Take care to get things correct before running `npm publish`!

> If releasing a "prerelease" of Fleet, run `npm publish --tag prerelease`. This way, you can publish a prerelease of fleetctl while the most recent fleetctl npm package, available for public download, is still the latest *official* release.

5. Announce the release in the #fleet channel of [osquery Slack](https://osquery.slack.com/join/shared_invite/zt-h29zm0gk-s2DBtGUTW4CFel0f0IjTEw#/) and update the channel's topic with the link to this release. Using `@here` requires admin permissions, so typically this announcement will be done by `@zwass`.

Announce the release via blog post (on Medium) and Twitter (linking to blog post).
