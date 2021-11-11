# Releasing Fleet

Note: Please prefix versions with `fleet-v` (eg. `fleet-v4.0.0`) in git tags, Helm charts, and NPM configs.

1. Update the [CHANGELOG](../../CHANGELOG.md) with the changes that have been made since the last
   Fleet release. Use `make changelog` to pull the changes files into `CHANGELOG.md`, then manually
   edit. When editing, order the most relevant/important changes at the time, and try to make the
   tone and syntax of the written language match throughout. `make changelog` will stage all changes
   file entries for deletion with the commit.

Update the NPM [package.json](../../tools/fleetctl-npm/package.json) with the new version number (do
not yet `npm publish`). Update the [Helm chart](../../charts/fleet/Chart.yaml) and [values
file](../../charts/fleet/values.yaml) with the new version number.

Commit these changes via Pull Request and pull the changes on the `main` branch locally. Check that
`HEAD` of the `main` branch points to the commit with these changes.

2. Tag and push the new release in Git:

```shell
git tag fleet-v<VERSION>
git push origin fleet-v<VERSION>
```

Note that `origin` may be `upstream` depending on your `git remote` configuration. The intent here
is to push the new tag to the `github.com/fleetdm/fleet` repository.

GitHub Actions will automatically begin building the new release after the tag is pushed.

---

Wait while GitHub Actions creates and uploads the artifacts...

---

When the Actions Workflow has completed:

3. Edit the draft release on the [GitHub releases page](https://github.com/fleetdm/fleet/releases).
   Use the version number as the release title. Use the below template for the release description
   (replace items in <> with the appropriate values):

````
### Changes

<COPY FROM CHANGELOG>

### Upgrading

Please visit our [update guide](https://fleetdm.com/docs/using-fleet/updating-fleet) for upgrade instructions.

### Documentation

Documentation for this release can be found at https://github.com/fleetdm/fleet/blob/<VERSION>/docs/README.md

### Binary Checksum

**SHA256**
```
<COPY FROM checksums.txt>
```
````

When editing is complete, publish the release.

4. Publish the new version of `fleetctl` on NPM. Run `npm publish` in the
   [fleetctl-npm](../../tools/fleetctl-npm/) directory. Note that NPM does not allow replacing a
   package without creating a new version number. Take care to get things correct before running
   `npm publish`!

> If releasing a "prerelease" of Fleet, run `npm publish --tag prerelease`. This way, you can
> publish a prerelease of fleetctl while the most recent fleetctl npm package, available for public
> download, is still the latest *official* release.

5. Announce the release in the #fleet channel of [osquery
   Slack](https://osquery.slack.com/join/shared_invite/zt-h29zm0gk-s2DBtGUTW4CFel0f0IjTEw#/) and
   update the channel's topic with the link to this release. Using `@here` requires admin
   permissions, so typically this announcement will be done by `@zwass`.

Announce the release via blog post (on Medium) and Twitter (linking to blog post).

## Patch releases

Generally, a patch should be released when bugs or performance issues are identified that prevent
users from getting their job done with Fleet.

### Process

#### The easy way

If all commits on `main` are acceptable for a patch (no high-risk changes, new features, etc.), then
the process is easy. Just follow the regular release process as described above, incrementing
only the patch (`major.minor.patch`) of the version number. In this scenario, there is no need to
perform any of the steps below.

#### The hard way

When only some of the newer changes in `main` are acceptable for release, a separate patch branch
must be created and relevant changes cherry-picked onto that branch:

1. Create the new branch, starting from the git tag of the prior release. Patch branches should be
   prefixed with `patch-`. In this example we are creating `4.3.1`:

   ```
   git checkout fleet-v4.3.0
   git checkout --branch patch-fleet-v4.3.1
   ```

2. Cherry pick the necessary commits into the new branch:
   
   ```
   git cherry-pick d34db33f
   ```

3. Push the branch to github.com/fleetdm/fleet:

   ```
   git push origin patch-fleet-v4.3.1
   ```

   When a `patch-*` branch is pushed, the [Docker publish
   Action](https://github.com/fleetdm/fleet/actions/workflows/goreleaser-snapshot-fleet.yaml) will
   be invoked to push a container image for QA with `fleetctl preview` (eg. `fleetctl preview
   --tag patch-fleet-v4.3.1`).

4. Check in the GitHub UI that Actions ran successfully for this branch and perform [QA smoke
   testing](../../.github/ISSUE_TEMPLATE/smoke-tests.md).

5. Follow the standard release instructions at the top of this document. Be sure that modifications
   to the changelog and config files are commited _on the `patch-*` branch_. When the patch has been
   released, return to finish the following steps.

6. Cherry-pick the commit containing the changelog updates into a new branch, and merge that commit
   into `main` through a Pull Request.

7. **Important!** Manually check the database migrations. Any migrations that are not cherry-picked in a
   patch must have a _higher_ timestamp than migrations that were cherry-picked. If there
   are new migrations that were not cherry-picked, verify that those migrations have higher
   timestamps. If they do not, submit a new Pull Request to increase the timestamps and ensure that
   migrations are run in the appropriate order.

   TODO [#2850](https://github.com/fleetdm/fleet/issues/2850): Improve docs/tooling for this.