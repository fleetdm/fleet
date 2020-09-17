Releasing Fleet
===============

1. Update the [CHANGELOG](/CHANGELOG.md) with the changes that have been made since the last Fleet release.

2. Tag and push the new release in Git:

``` shell
git tag <VERSION>
git push origin <VERSION>
```

3. Build the new binary bundle (ensure working tree is clean because this will effect the version string built into the binary):

``` shell
make binary-bundle
```

4. Create a new release on the [GitHub releases page](https://github.com/kolide/fleet/releases). Select the newly pushed tag (GitHub should say "Existing tag"). Use the version number as the release title. Use the below template for the release description (replace items in <> with the appropriate values):

````
### Changes

<COPY FROM CHANGELOG>

### Upgrading

Please visit our [update guide](https://github.com/kolide/fleet/blob/master/docs/infrastructure/updating-fleet.md) for upgrade instructions.

### Documentation

Documentation for this release can be found at https://github.com/kolide/fleet/blob/<VERSION>/docs/README.md

### Binary Checksum

```
sha256sum fleet.zip
<HASH VALUE>  fleet.zip
```

````

Upload the `fleet.zip` binary bundle and click "Publish Release".

5. Push the new version to Docker Hub (ensure working tree is clean because this will effect the version string built into the binary):

``` shell
make docker-push-release
```

6. Announce the release in the #kolide channel of [osquery Slack](https://osquery.slack.com/join/shared_invite/zt-h29zm0gk-s2DBtGUTW4CFel0f0IjTEw#/).
